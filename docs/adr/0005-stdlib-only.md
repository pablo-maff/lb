# ADR 0005: Zero external dependencies

**Status:** Accepted

## Context

The LB needs HTTP proxying, concurrency primitives, TCP dialing, CLI flags, and a background ticker. All are available in the Go standard library.

## Decision

No external dependencies. `go.mod` declares only the module name and Go version.

## Consequences

- Single static binary with no `vendor/` or module cache required at deploy time.
- No dependency supply-chain risk or version drift.
- Ceiling: stdlib `httputil.ReverseProxy` has no built-in circuit breaker, weighted routing, or TLS termination. Adding any of those would likely require external packages.
