package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
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

	for _, tok := range strings.Split(serverList, ",") {
		serverUrl, err := url.Parse(strings.TrimSpace(tok))
		if err != nil {
			log.Fatalf("invalid backend URL %q: %v", tok, err)
		}

		proxy := httputil.NewSingleHostReverseProxy(serverUrl)
		proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, e error) {
			log.Printf("[%s] error: %v\n", serverUrl.Host, e)
			retries := GetRetryFromContext(r)
			if retries < 3 {
				select {
				case <-time.After(10 * time.Millisecond):
					ctx := context.WithValue(r.Context(), Retry, retries+1)
					proxy.ServeHTTP(w, r.WithContext(ctx))
				}
				return
			}
			serverPool.MarkBackendStatus(serverUrl, false)
			attempts := GetAttemptsFromContext(r)
			log.Printf("%s(%s) retrying, attempt %d\n", r.RemoteAddr, r.URL.Path, attempts)
			ctx := context.WithValue(r.Context(), Attempts, attempts+1)
			lb(w, r.WithContext(ctx))
		}

		serverPool.AddBackend(&Backend{
			URL:          serverUrl,
			Alive:        true,
			ReverseProxy: proxy,
		})
		log.Printf("Configured backend: %s\n", serverUrl)
	}

	server := http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: http.HandlerFunc(lb),
	}

	go healthCheck()

	log.Printf("Load Balancer started at :%d\n", port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
