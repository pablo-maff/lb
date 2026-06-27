# ADR 0002: Atomic counter for round-robin selection

**Status:** Accepted

## Context

Every incoming request reads and increments a shared counter to select the next backend. Under concurrent load this counter is a hot write path.

## Decision

Use `sync/atomic.AddUint64` to increment the counter and `atomic.StoreUint64` to reset it when skipping dead backends. No mutex around the counter itself.

## Consequences

- Lock-free increment; no goroutine contention on the counter under high concurrency.
- Counter wraps at `uint64` max (~1.8×10¹⁹) — effectively never in practice.
- Selection is not perfectly fair when backends are skipped (dead peers), but close enough for a simple LB. A stricter fair-share algorithm would need a mutex anyway.
