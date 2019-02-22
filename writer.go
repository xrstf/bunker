package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	writerTTL = 1 * time.Minute
)

type writer struct {
	file    *os.File
	expires time.Time
}

func NewWriter(path string) (*writer, error) {
	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory %s: %v", directory, err)
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s for appending: %v", path, err)
	}

	return &writer{
		file:    f,
		expires: time.Now().Add(writerTTL),
	}, nil
}

func (w *writer) Close() error {
	var err error

	if w.file != nil {
		err = w.file.Close()
		w.file = nil
	}

	return err
}

func (w *writer) Write(record Record) error {
	encoder := json.NewEncoder(w.file)
	if err := encoder.Encode(record); err != nil {
		return fmt.Errorf("failed to write record: %v", err)
	}

	return nil
}

func (w *writer) Expired(now time.Time) bool {
	return now.IsZero() || now.After(w.expires)
}

func (w *writer) Touch() {
	w.expires = time.Now().Add(writerTTL)
}
