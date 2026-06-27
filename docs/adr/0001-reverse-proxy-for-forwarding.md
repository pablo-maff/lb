# ADR 0001: Use httputil.ReverseProxy for request forwarding

**Status:** Accepted

## Context

Requests arriving at the LB must be forwarded to a backend and the response relayed back. This could be done manually (read body, copy headers, make outbound request, write response) or via a stdlib abstraction.

## Decision

Use `httputil.ReverseProxy` per backend. It handles header rewriting, hop-by-hop stripping, body copying, and flush semantics correctly. Its `ErrorHandler` hook gives us a clean injection point for retry and failover logic without wrapping the proxy in middleware.

## Consequences

- No manual HTTP relay code to maintain or get wrong.
- One `ReverseProxy` instance per backend — constructed once at startup, reused for every request routed to that backend.
- Failover logic is coupled to the proxy's `ErrorHandler`; adding per-request middleware requires wrapping the proxy's `ServeHTTP`.
