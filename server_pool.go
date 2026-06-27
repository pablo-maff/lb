package main

import (
	"log"
	"net/url"
	"sync/atomic"
)

type ServerPool struct {
	backends []*Backend
	current  uint64
}

func (s *ServerPool) AddBackend(b *Backend) {
	s.backends = append(s.backends, b)
}

func (s *ServerPool) NextIndex() int {
	return int(atomic.AddUint64(&s.current, 1) % uint64(len(s.backends)))
}

func (s *ServerPool) MarkBackendStatus(u *url.URL, alive bool) {
	for _, b := range s.backends {
		if b.URL.String() == u.String() {
			b.SetAlive(alive)
			break
		}
	}
}

// GetNextPeer does a full cycle from the next index looking for an alive backend.
func (s *ServerPool) GetNextPeer() *Backend {
	if len(s.backends) == 0 {
		return nil
	}
	next := s.NextIndex()
	l := len(s.backends)
	for i := 0; i < l; i++ {
		idx := (next + i) % l
		if s.backends[idx].IsAlive() {
			atomic.StoreUint64(&s.current, uint64(idx))
			return s.backends[idx]
		}
	}
	return nil
}

func (s *ServerPool) HealthCheck() {
	for _, b := range s.backends {
		alive := isBackendAlive(b.URL)
		b.SetAlive(alive)
		status := "up"
		if !alive {
			status = "down"
		}
		log.Printf("%s [%s]\n", b.URL, status)
	}
}
