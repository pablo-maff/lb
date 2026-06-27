package balancer

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"testing"
)

func mustURL(raw string) *url.URL {
	u, _ := url.Parse(raw)
	return u
}

// -- ServerPool --

func TestNextIndexWraps(t *testing.T) {
	sp := &ServerPool{}
	sp.AddBackend(&Backend{URL: mustURL("http://a"), Alive: true})
	sp.AddBackend(&Backend{URL: mustURL("http://b"), Alive: true})

	first := sp.NextIndex()
	second := sp.NextIndex()
	if second == first {
		t.Fatalf("NextIndex should advance: got %d twice", first)
	}
}

func TestGetNextPeerSkipsDead(t *testing.T) {
	sp := &ServerPool{}
	sp.AddBackend(&Backend{URL: mustURL("http://dead"), Alive: false})
	sp.AddBackend(&Backend{URL: mustURL("http://live"), Alive: true})

	peer := sp.GetNextPeer()
	if peer == nil {
		t.Fatal("expected a live peer")
	}
	if peer.URL.String() != "http://live" {
		t.Fatalf("expected http://live, got %s", peer.URL)
	}
}

func TestGetNextPeerAllDead(t *testing.T) {
	sp := &ServerPool{}
	sp.AddBackend(&Backend{URL: mustURL("http://a"), Alive: false})
	if sp.GetNextPeer() != nil {
		t.Fatal("expected nil when all backends dead")
	}
}

func TestMarkBackendStatus(t *testing.T) {
	sp := &ServerPool{}
	u := mustURL("http://a")
	sp.AddBackend(&Backend{URL: u, Alive: true})
	sp.MarkBackendStatus(u, false)
	if sp.backends[0].IsAlive() {
		t.Fatal("expected backend to be marked dead")
	}
}

func TestHealthCheckMarksDeadBackend(t *testing.T) {
	sp := &ServerPool{}
	sp.AddBackend(&Backend{URL: mustURL("http://127.0.0.1:1"), Alive: true})
	sp.HealthCheck()
	if sp.backends[0].IsAlive() {
		t.Fatal("unreachable backend should be marked dead")
	}
}

// -- ServeHTTP --

func TestServeHTTP503NoBackends(t *testing.T) {
	sp := &ServerPool{}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	sp.ServeHTTP(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}
}

func TestServeHTTP503AllDead(t *testing.T) {
	sp := &ServerPool{}
	sp.AddBackend(&Backend{URL: mustURL("http://dead"), Alive: false})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	sp.ServeHTTP(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}
}

func TestServeHTTPProxiesToLiveBackend(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	u, _ := url.Parse(upstream.URL)
	sp := &ServerPool{}
	sp.AddBackend(&Backend{URL: u, Alive: true, ReverseProxy: httputil.NewSingleHostReverseProxy(u)})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	sp.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestServeHTTP503AfterMaxAttempts(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer upstream.Close()

	u, _ := url.Parse(upstream.URL)
	sp := &ServerPool{}
	sp.AddBackend(&Backend{URL: u, Alive: true, ReverseProxy: httputil.NewSingleHostReverseProxy(u)})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := context.WithValue(req.Context(), attempts, 4)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	sp.ServeHTTP(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 after max attempts, got %d", w.Code)
	}
}
