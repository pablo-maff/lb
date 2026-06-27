# Project Summary

Built a round-robin HTTP load balancer in Go from scratch, following the [kasvith tutorial](https://kasvith.me/posts/lets-create-a-simple-lb-go/).

## Implementation

- `Backend` struct with thread-safe alive flag (`sync.RWMutex`)
- `ServerPool` with atomic round-robin counter (`sync/atomic`) and peer selection that skips dead backends
- `httputil.ReverseProxy` per backend — no manual HTTP relay
- Active failover via `proxy.ErrorHandler`: 3 retries with 10ms sleep, then marks backend dead and reroutes
- Passive health check: TCP dial every 20s via a goroutine ticker
- Retry/attempt counts carried through `http.Request.Context` to avoid global state
- `-backends` and `-port` CLI flags

## Testing

- Unit tests for `Backend` alive-flag concurrency, `ServerPool` round-robin and peer selection, passive health check, and the `ServeHTTP` handler (503 paths, proxy-to-live-backend, max-attempts cap)
- Integration tests (`-tags integration`) that run the actual binary over the network: round-robin distribution, single-backend failover, and all-dead 503

## Bugs caught in the process

- Divide-by-zero panic in `GetNextPeer` when pool is empty — fixed with an early nil return
- Single-case `select { case <-time.After(...) }` flagged by staticcheck — replaced with `time.Sleep`

## Structure

Refactored from a flat `package main` to:

```
cmd/lb/             — entry point (~35 lines)
internal/balancer/  — all logic (Backend, ServerPool, proxy wiring)
test/               — integration tests
```

## Docs

- 5 ADRs covering: ReverseProxy choice, atomic counter, two-layer health checking, context for retry state, zero external dependencies
- draw.io diagram with 4 pages: request flow, passive health check, data model, package dependency graph
- README with structure, run instructions, and test commands
