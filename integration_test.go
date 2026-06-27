//go:build integration

package main

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

var lbBin string

func TestMain(m *testing.M) {
	bin := "/tmp/lb-integration-bin"
	out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "build failed: %s\n", out)
		os.Exit(1)
	}
	lbBin = bin
	code := m.Run()
	os.Remove(bin)
	os.Exit(code)
}

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("freePort: %v", err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return port
}

func startLB(t *testing.T, backends []string, port int) *exec.Cmd {
	t.Helper()
	cmd := exec.Command(lbBin,
		"-backends", strings.Join(backends, ","),
		"-port", fmt.Sprintf("%d", port),
	)
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("start lb: %v", err)
	}
	t.Cleanup(func() { cmd.Process.Kill() })

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), 100*time.Millisecond)
		if err == nil {
			conn.Close()
			return cmd
		}
		time.Sleep(30 * time.Millisecond)
	}
	t.Fatal("lb did not become ready in time")
	return nil
}

func get(t *testing.T, port int) int {
	t.Helper()
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/", port))
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	resp.Body.Close()
	return resp.StatusCode
}

// TestIntegration_RoundRobin verifies requests are distributed across both backends.
func TestIntegration_RoundRobin(t *testing.T) {
	var hits [2]atomic.Int64
	backends := [2]*httptest.Server{
		httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hits[0].Add(1) })),
		httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hits[1].Add(1) })),
	}
	defer backends[0].Close()
	defer backends[1].Close()

	port := freePort(t)
	startLB(t, []string{backends[0].URL, backends[1].URL}, port)

	for i := 0; i < 10; i++ {
		if code := get(t, port); code != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d", i, code)
		}
	}

	if hits[0].Load() == 0 || hits[1].Load() == 0 {
		t.Fatalf("expected both backends to be hit, got hits: %d, %d", hits[0].Load(), hits[1].Load())
	}
}

// TestIntegration_Failover verifies the LB keeps serving when one backend dies.
func TestIntegration_Failover(t *testing.T) {
	alive := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	dead := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer alive.Close()

	port := freePort(t)
	startLB(t, []string{dead.URL, alive.URL}, port)

	// confirm both serve
	if code := get(t, port); code != http.StatusOK {
		t.Fatalf("pre-kill: expected 200, got %d", code)
	}

	dead.Close() // kill one backend

	// allow active failover to mark it dead (up to 3 retries × 10ms each)
	time.Sleep(200 * time.Millisecond)

	for i := 0; i < 5; i++ {
		if code := get(t, port); code != http.StatusOK {
			t.Fatalf("post-kill request %d: expected 200, got %d", i, code)
		}
	}
}

// TestIntegration_AllDead verifies a 503 when no backends are reachable.
func TestIntegration_AllDead(t *testing.T) {
	b1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	b2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	port := freePort(t)
	startLB(t, []string{b1.URL, b2.URL}, port)

	b1.Close()
	b2.Close()

	time.Sleep(200 * time.Millisecond)

	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/", port))
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", resp.StatusCode)
	}
}
