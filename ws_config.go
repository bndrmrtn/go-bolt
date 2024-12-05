package gale

import (
	"time"
)

// WSConfig is the configuration of the websocket
type WSConfig struct {
	ReadTimeout         time.Duration
	MessageBufferSize   int
	MaxConcurrentReads  int
	MaxConcurrentWrites int
}

func (w *WSConfig) check() {
	if w.ReadTimeout == 0 {
		w.ReadTimeout = time.Second * 10
	}

	if w.MessageBufferSize == 0 {
		w.MessageBufferSize = 100
	}

	if w.MaxConcurrentReads == 0 {
		w.MaxConcurrentReads = 10
	}

	if w.MaxConcurrentWrites == 0 {
		w.MaxConcurrentWrites = 10
	}
}

func defaultWSConfig() *WSConfig {
	return &WSConfig{
		ReadTimeout:         time.Second * 10,
		MessageBufferSize:   100,
		MaxConcurrentReads:  10,
		MaxConcurrentWrites: 10,
	}
}
