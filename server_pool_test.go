package main

import (
	"net/url"
	"testing"
)

func mustURL(raw string) *url.URL {
	u, _ := url.Parse(raw)
	return u
}

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
	dead := &Backend{URL: mustURL("http://dead"), Alive: false}
	live := &Backend{URL: mustURL("http://live"), Alive: true}
	sp.AddBackend(dead)
	sp.AddBackend(live)

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
