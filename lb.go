package main

import (
	"log"
	"net"
	"net/http"
	"net/url"
	"time"
)

type contextKey int

const (
	Attempts contextKey = iota
	Retry
)

func isBackendAlive(u *url.URL) bool {
	conn, err := net.DialTimeout("tcp", u.Host, 2*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func GetAttemptsFromContext(r *http.Request) int {
	if v, ok := r.Context().Value(Attempts).(int); ok {
		return v
	}
	return 1
}

func GetRetryFromContext(r *http.Request) int {
	if v, ok := r.Context().Value(Retry).(int); ok {
		return v
	}
	return 0
}

var serverPool ServerPool

func lb(w http.ResponseWriter, r *http.Request) {
	attempts := GetAttemptsFromContext(r)
	if attempts > 3 {
		log.Printf("%s(%s) max attempts reached\n", r.RemoteAddr, r.URL.Path)
		http.Error(w, "Service not available", http.StatusServiceUnavailable)
		return
	}
	peer := serverPool.GetNextPeer()
	if peer != nil {
		peer.ReverseProxy.ServeHTTP(w, r)
		return
	}
	http.Error(w, "Service not available", http.StatusServiceUnavailable)
}

func healthCheck() {
	t := time.NewTicker(20 * time.Second)
	for range t.C {
		log.Println("Starting health check...")
		serverPool.HealthCheck()
		log.Println("Health check completed")
	}
}
