package main

import (
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

type sink struct {
	config       Config
	logger       logrus.FieldLogger
	jobs         chan interface{}
	lock         sync.RWMutex
	writers      map[string]*writer
	workerAlive  chan struct{}
	gcKillswitch chan struct{}
	gcAlive      chan struct{}
}

func NewSink(config Config, logger logrus.FieldLogger) (*sink, error) {
	return &sink{
		config:       config,
		logger:       logger,
		jobs:         make(chan interface{}, 10),
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
	for _, record := range payload {
		s.jobs <- recordJob{record}
	}

	return len(payload)
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
	s.lock.Lock()

	writer, ok := s.writers[path]
	if ok {
		writer.Close()
		delete(s.writers, path)
	}

	s.lock.Unlock()
}

func (s *sink) closeExpiredWriters() {
	s.closeWritersBy(time.Now())
}

func (s *sink) closeAllWriters() {
	s.closeWritersBy(time.Time{})
}

func (s *sink) closeWritersBy(t time.Time) {
	s.lock.RLock()

	for path, writer := range s.writers {
		if writer.Expired(t) {
			s.jobs <- closeWriterJob{path}
		}
	}

	s.lock.RUnlock()
}
