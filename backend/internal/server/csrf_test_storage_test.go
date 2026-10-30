package server

import (
	"sync"
	"time"
)

type csrfTestStorage struct {
	mu     sync.Mutex
	values map[string][]byte
}

func (s *csrfTestStorage) Get(key string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.values[key], nil
}
func (s *csrfTestStorage) Set(key string, value []byte, _ time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values[key] = value
	return nil
}
func (s *csrfTestStorage) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.values, key)
	return nil
}
func (s *csrfTestStorage) Reset() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values = map[string][]byte{}
	return nil
}
func (s *csrfTestStorage) Close() error { return nil }
