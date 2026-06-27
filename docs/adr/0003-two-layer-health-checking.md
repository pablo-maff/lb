# ADR 0003: Two-layer health checking (active failover + passive TCP probe)

**Status:** Accepted

## Context

A backend can fail mid-flight (active failure) or quietly stop accepting connections between requests (silent failure). A single mechanism handles only one case well.

## Decision

Two independent layers:

1. **Active failover** — `proxy.ErrorHandler` retries a failed request up to 3 times on the same backend (10 ms apart), then marks it dead and reroutes to the next available peer via a recursive `lb()` call with an incremented attempt counter. Caps at 3 total attempts before returning 503.

2. **Passive TCP probe** — a background goroutine ticks every 20 seconds and TCP-dials every backend (`net.DialTimeout`, 2 s). Updates the alive flag regardless of traffic. This is the only mechanism that can resurrect a backend after it recovers.

## Consequences

- Active failover catches failures immediately with no polling lag.
- Passive probe catches silent failures and recovers previously-dead backends — neither of which the active layer handles.
- A backend that fails exactly at request time may serve one error before the active layer marks it down.
- 20-second probe interval means a recovered backend is invisible to the LB for up to 20 s.
