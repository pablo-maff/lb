package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

	"lb/internal/balancer"
)

func main() {
	var serverList string
	var port int
	flag.StringVar(&serverList, "backends", "", "Comma-separated backend URLs, e.g. http://localhost:3031,http://localhost:3032")
	flag.IntVar(&port, "port", 3030, "Port to serve on")
	flag.Parse()

	if serverList == "" {
		log.Fatal("Please provide one or more backends with -backends")
	}

	pool := &balancer.ServerPool{}
	for _, tok := range strings.Split(serverList, ",") {
		u, err := url.Parse(strings.TrimSpace(tok))
		if err != nil {
			log.Fatalf("invalid backend URL %q: %v", tok, err)
		}
		pool.AddBackend(&balancer.Backend{
			URL:          u,
			Alive:        true,
			ReverseProxy: pool.NewProxy(u),
		})
		log.Printf("Configured backend: %s\n", u)
	}

	go pool.HealthCheckLoop()
	log.Printf("Load Balancer started at :%d\n", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), pool))
}
