# lb

A simple round-robin HTTP load balancer in Go. No external dependencies.

## Run

```bash
go build -o lb .
./lb -backends "http://localhost:3031,http://localhost:3032" -port 3030
```

| Flag | Default | Description |
|------|---------|-------------|
| `-backends` | required | Comma-separated list of backend URLs |
| `-port` | `3030` | Port to listen on |

## How it works

Incoming requests are distributed across backends in round-robin order. Dead backends are skipped automatically.

**Active failover:** if a backend errors, the request is retried up to 3 times on that backend, then rerouted to the next available peer. After 3 reroutes with no success, the client receives a `503`.

**Passive health check:** every 20 seconds, each backend is TCP-dialled. Backends that recover are automatically returned to rotation.

See [`docs/diagram.md`](docs/diagram.md) for a visual breakdown and [`docs/adr/`](docs/adr/) for architecture decisions.

## Test

```bash
go test ./...
```
