// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0

// Package processlog writes bounded, structured lifecycle events for a render run.
package processlog

import (
	"encoding/json"
	"io"
	"sync"
	"time"
)

type Event struct {
	Time          time.Time `json:"time"`
	Phase         string    `json:"phase"`
	Outcome       string    `json:"outcome"`
	InputProfile  string    `json:"input_profile,omitempty"`
	ReportProfile string    `json:"report_profile,omitempty"`
	Detail        string    `json:"detail,omitempty"`
	ErrorCode     string    `json:"error_code,omitempty"`
}

type Logger struct {
	mu sync.Mutex
	w  io.Writer
}

func New(w io.Writer) *Logger { return &Logger{w: w} }

func (logger *Logger) Event(event Event) error {
	logger.mu.Lock()
	defer logger.mu.Unlock()
	return json.NewEncoder(logger.w).Encode(event)
}
