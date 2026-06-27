# Diagrams

See [`diagram.drawio`](diagram.drawio) — open with [draw.io](https://app.diagrams.net) or the VS Code draw.io extension.

Four pages:

| Page | Contents |
|------|----------|
| Request Flow | Client → lb handler → round-robin → ReverseProxy → backends, with ErrorHandler retry/failover loop |
| Passive Health Check | 20s ticker → TCP dial each backend → SetAlive |
| Data Model | ServerPool and Backend structs with fields and methods |
| Package Dependencies | cmd/lb → internal/balancer → Go stdlib imports |
