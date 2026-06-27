package balancer

import (
	"context"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync/atomic"
	"time"
)

type contextKey int

const (
	attempts contextKey = iota
	retry
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

func (s *ServerPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if getAttemptsFromContext(r) > 3 {
		log.Printf("%s(%s) max attempts reached\n", r.RemoteAddr, r.URL.Path)
		http.Error(w, "Service not available", http.StatusServiceUnavailable)
		return
	}
	peer := s.GetNextPeer()
	if peer != nil {
		peer.ReverseProxy.ServeHTTP(w, r)
		return
	}
	http.Error(w, "Service not available", http.StatusServiceUnavailable)
}

// NewProxy creates a reverse proxy for u with active failover wired to this pool.
func (s *ServerPool) NewProxy(u *url.URL) *httputil.ReverseProxy {
	proxy := httputil.NewSingleHostReverseProxy(u)
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, e error) {
		log.Printf("[%s] error: %v\n", u.Host, e)
		retries := getRetryFromContext(r)
		if retries < 3 {
			select {
			case <-time.After(10 * time.Millisecond):
				ctx := context.WithValue(r.Context(), retry, retries+1)
				proxy.ServeHTTP(w, r.WithContext(ctx))
			}
			return
		}
		s.MarkBackendStatus(u, false)
		a := getAttemptsFromContext(r)
		log.Printf("%s(%s) retrying, attempt %d\n", r.RemoteAddr, r.URL.Path, a)
		ctx := context.WithValue(r.Context(), attempts, a+1)
		s.ServeHTTP(w, r.WithContext(ctx))
	}
	return proxy
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

func (s *ServerPool) HealthCheckLoop() {
	t := time.NewTicker(20 * time.Second)
	for range t.C {
		log.Println("Starting health check...")
		s.HealthCheck()
		log.Println("Health check completed")
	}
}

func getAttemptsFromContext(r *http.Request) int {
	if v, ok := r.Context().Value(attempts).(int); ok {
		return v
	}
	return 1
}

func getRetryFromContext(r *http.Request) int {
	if v, ok := r.Context().Value(retry).(int); ok {
		return v
	}
	return 0
}

func isBackendAlive(u *url.URL) bool {
	conn, err := net.DialTimeout("tcp", u.Host, 2*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}
