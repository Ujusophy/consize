package audit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Record struct {
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
	Payload   any       `json:"payload"`
}

type Sink interface {
	Write(recordType string, payload any) error
}

type NoopSink struct{}

func (NoopSink) Write(string, any) error { return nil }

type JSONLSink struct {
	path string
}

func NewJSONLSink(path string) (*JSONLSink, error) {
	if path == "" {
		return nil, fmt.Errorf("audit path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	return &JSONLSink{path: path}, nil
}

func (s *JSONLSink) Write(recordType string, payload any) error {
	f, err := os.OpenFile(s.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	record := Record{Type: recordType, Timestamp: time.Now().UTC(), Payload: payload}
	enc := json.NewEncoder(f)
	return enc.Encode(record)
}
