package main

import (
	"sync"
	"testing"
)

func TestSetAndIsAlive(t *testing.T) {
	b := &Backend{}
	b.SetAlive(true)
	if !b.IsAlive() {
		t.Fatal("expected alive=true")
	}
	b.SetAlive(false)
	if b.IsAlive() {
		t.Fatal("expected alive=false")
	}
}

func TestIsAliveConcurrent(t *testing.T) {
	b := &Backend{Alive: true}
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			b.SetAlive(true)
			_ = b.IsAlive()
		}()
	}
	wg.Wait()
}
