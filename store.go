package gale

import (
	"errors"
	"sync"
	"time"
)

// SessionStore is an interface for session storage.
type SessionStore interface {
	Get(key string) ([]byte, error)
	Exists(key string) bool
	Set(key string, value []byte) error
	SetEx(key string, value []byte, expiry time.Duration) error
	Del(key string) error
}

// MemoryStore is an in-memory session storage.
type MemoryStore struct {
	data       map[string]memStoreEntry
	mu         sync.RWMutex
	gcInterval time.Duration
	done       chan struct{}
}

type memStoreEntry struct {
	data []byte
	exp  *time.Time
}

// NewMemStorage creates a new MemoryStore.
func NewMemStorage(gcInterval ...time.Duration) SessionStore {
	var duration time.Duration
	if len(gcInterval) > 0 {
		duration = gcInterval[0]
	} else {
		duration = time.Second * 15
	}

	m := &MemoryStore{
		data:       make(map[string]memStoreEntry),
		gcInterval: duration,
		mu:         sync.RWMutex{},
		done:       make(chan struct{}),
	}
	go m.gc()
	return m
}

func (s *MemoryStore) Close() error {
	s.done <- struct{}{}
	return nil
}

func (s *MemoryStore) Get(key string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if v, ok := s.data[key]; ok {
		return v.data, nil
	}
	return nil, errors.New("key not found")
}

func (s *MemoryStore) Exists(key string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, ok := s.data[key]
	return ok
}

func (s *MemoryStore) Set(key string, value []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data[key] = memStoreEntry{data: value}
	return nil
}

func (s *MemoryStore) SetEx(key string, value []byte, expiry time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	exp := time.Now().Add(expiry)
	s.data[key] = memStoreEntry{data: value, exp: &exp}
	return nil
}

func (s *MemoryStore) Del(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.data, key)
	return nil
}

func (s *MemoryStore) gc() {
	ticker := time.NewTicker(s.gcInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.done:
			return
		case <-ticker.C:
			for k, v := range s.data {
				if v.exp != nil && v.exp.Before(time.Now()) {
					s.mu.Lock()
					delete(s.data, k)
					s.mu.Unlock()
				}
			}
		}
	}
}
