package main

import (
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

type sink struct {
	config       Config
	filter       *filter
	logger       logrus.FieldLogger
	jobs         chan interface{}
	lock         sync.RWMutex
	writers      map[string]*writer
	workerAlive  chan struct{}
	gcKillswitch chan struct{}
	gcAlive      chan struct{}
}

func NewSink(config Config, filter *filter, logger logrus.FieldLogger) (*sink, error) {
	return &sink{
		config:       config,
		filter:       filter,
		logger:       logger,
		jobs:         make(chan interface{}, 10000),
		lock:         sync.RWMutex{},
		writers:      make(map[string]*writer),
		workerAlive:  make(chan struct{}),
		gcKillswitch: make(chan struct{}),
		gcAlive:      make(chan struct{}),
	}, nil
}

// ProcessQueue is meant to run as a separate goroutine
// and processes the job queue, i.e. it writes records
// and handling close requests for expired file writers.
func (s *sink) ProcessQueue() {
	for job := range s.jobs {
		switch j := job.(type) {
		case recordJob:
			s.handleRecord(j.record)

		case closeWriterJob:
			s.closeWriter(j.path)
		}
	}

	close(s.workerAlive)
}

// GarbageCollect is meant to run as a separate goroutine
// and takes care of closing any expired, i.e. unused,
// file writers. This goroutine ends when you call Close().
func (s *sink) GarbageCollect() {
	defer close(s.gcAlive)

	for {
		select {
		case <-s.gcKillswitch:
			return

		case <-time.After(5 * time.Minute):
			s.closeExpiredWriters()
		}
	}
}

func (s *sink) AddPayload(payload Payload) int {
	s.logger.Debugf("Adding payload (len=%d) ...", len(payload))

	num := 0

	for _, record := range payload {
		if s.filter == nil || s.filter.IncludeRecord(record) {
			s.jobs <- recordJob{record}
			num++
		}
	}

	s.logger.Debug("Done adding payload.")

	return num
}

// Close stops the garbage collection and the queue processor
// goroutines and waits for both to end. It will also take
// care of closing all opened files.
func (s *sink) Close() {
	// stop the garbage collection routine
	close(s.gcKillswitch)
	<-s.gcAlive

	// close all writers
	s.closeAllWriters()

	// stop accepting new jobs and wait until all have been processed
	close(s.jobs)
	<-s.workerAlive
}

func (s *sink) handleRecord(record Record) {
	// build final file path
	replacer := strings.NewReplacer(record.StringReplacements()...)
	path := filepath.Join(s.config.Target, replacer.Replace(s.config.Pattern))

	// attempt to find an existing writer
	s.lock.Lock()

	writer, ok := s.writers[path]
	if !ok {
		writer, _ = NewWriter(path)
		s.writers[path] = writer
	}

	s.lock.Unlock()

	writer.Write(record)
}

func (s *sink) closeWriter(path string) {
	s.logger.Debugf("Closing writer %s ...", path)
	s.lock.Lock()

	writer, ok := s.writers[path]
	if ok {
		writer.Close()
		delete(s.writers, path)
	}

	s.lock.Unlock()
	s.logger.Debug("Done closing writer.")
}

func (s *sink) closeExpiredWriters() {
	s.closeWritersBy(time.Now())
}

func (s *sink) closeAllWriters() {
	s.closeWritersBy(time.Time{})
}

func (s *sink) closeWritersBy(t time.Time) {
	s.logger.Debugf("Starting to close writers... (t = %v)", t)
	s.lock.RLock()

	for path, writer := range s.writers {
		if writer.Expired(t) {
			s.jobs <- closeWriterJob{path}
		}
	}

	s.lock.RUnlock()
	s.logger.Debug("Done closing writers.")
}

func (s *sink) Describe(descriptions chan<- *prometheus.Desc) {
	descriptions <- prometheus.NewDesc("bunker_open_writers_total", "Total number of currently open file writers", nil, nil)
}

func (s *sink) Collect(metrics chan<- prometheus.Metric) {
	s.logger.Debug("Collecting sink metrics...")

	s.lock.RLock()
	total := len(s.writers)
	s.lock.RUnlock()

	s.logger.Debug("Sink metrics collection done.")

	gauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "bunker_open_writers_total",
	})

	gauge.Set(float64(total))

	metrics <- gauge
}
