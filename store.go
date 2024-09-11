package bolt

import (
	"errors"
	"sync"
	"time"
)

type SessionStore interface {
	Get(key string) ([]byte, error)
	Exists(key string) bool
	Set(key string, value []byte) error
	SetEx(key string, value []byte, expiry time.Duration) error
	Del(key string) error
}

type MemoryStore struct {
	data map[string]memStoreEntry
	mu   sync.RWMutex
	done chan struct{}
}

type memStoreEntry struct {
	data []byte
	exp  *time.Time
}

func NewMemStorage() *MemoryStore {
	m := &MemoryStore{
		data: make(map[string]memStoreEntry),
		mu:   sync.RWMutex{},
		done: make(chan struct{}),
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
	ticker := time.NewTicker(time.Second * 30)
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
