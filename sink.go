package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type sink struct {
	config Config
}

func NewSink(config Config) *sink {
	return &sink{
		config: config,
	}
}

func (s *sink) StorePayload(p Payload) (int, error) {
	var err error

	for _, record := range p {
		err = s.storeRecord(record)
		if err != nil {
			break
		}
	}

	return len(p), err
}

func (s *sink) storeRecord(record Record) error {
	fullPath := s.recordPath(record)

	directory := filepath.Dir(fullPath)
	if err := os.MkdirAll(directory, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %v", directory, err)
	}

	f, err := os.OpenFile(fullPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("failed to open %s for appending: %v", fullPath, err)
	}
	defer f.Close()

	encoder := json.NewEncoder(f)
	if err := encoder.Encode(record); err != nil {
		return fmt.Errorf("failed to write record: %v", err)
	}

	return nil
}

func (s *sink) recordPath(record Record) string {
	replacer := strings.NewReplacer(record.StringReplacements()...)
	filename := replacer.Replace(s.config.Pattern)

	return filepath.Join(s.config.Target, filename)
}
