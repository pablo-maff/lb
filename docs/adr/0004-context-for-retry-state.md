# ADR 0004: Request context for retry and attempt state

**Status:** Accepted

## Context

The active failover path calls `lb()` recursively. Each recursive call needs to know how many retries and top-level attempts have already occurred to enforce the caps. This state must travel with the request without modifying function signatures or introducing global per-request state.

## Decision

Carry retry count and attempt count as values in `http.Request.Context`, keyed by an unexported `contextKey` int type. `GetRetryFromContext` and `GetAttemptsFromContext` provide typed reads with safe defaults (0 retries, 1 attempt).

## Consequences

- No changes to `http.Handler` or `http.HandlerFunc` signatures.
- State is scoped to the request lifecycle; no cleanup needed.
- Using an unexported type as the context key prevents key collisions with middleware higher in the stack.
- Recursive `lb()` calls create a new context per call — cheap allocation, no shared mutable state.
