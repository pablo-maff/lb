package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"testing"
)

func resetPool(backends ...*Backend) {
	serverPool = ServerPool{}
	for _, b := range backends {
		serverPool.AddBackend(b)
	}
}

func TestLbReturns503WhenNoBackends(t *testing.T) {
	resetPool()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	lb(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}
}

func TestLbReturns503WhenAllDead(t *testing.T) {
	resetPool(&Backend{URL: mustURL("http://dead"), Alive: false})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	lb(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}
}

func TestLbProxiesToLiveBackend(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	u, _ := url.Parse(upstream.URL)
	proxy := httputil.NewSingleHostReverseProxy(u)
	b := &Backend{URL: u, Alive: true, ReverseProxy: proxy}
	resetPool(b)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	lb(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestLbRejects503AfterMaxAttempts(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	u, _ := url.Parse(upstream.URL)
	proxy := httputil.NewSingleHostReverseProxy(u)
	b := &Backend{URL: u, Alive: true, ReverseProxy: proxy}
	resetPool(b)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := context.WithValue(req.Context(), Attempts, 4)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	lb(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 after max attempts, got %d", w.Code)
	}
}
