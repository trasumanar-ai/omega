# Server

Go backend for the Omega economic simulation.

## Run

```bash
# HTTP server (used by frontend)
go run ./cmd/serve -port 8090

# CLI experiment runner
go run ./cmd/experiment -steps 120 -sample-every 20
go run ./cmd/experiment experiment run -template default_algorithms
```

## Structure

```
server/
├── internal/econ/        # Simulation engine
│   ├── simulation.go     # Core tick loop
│   ├── types.go          # Config, snapshots, market types
│   └── openrouter.go     # LLM decision provider
├── cmd/serve/            # HTTP API server
└── cmd/experiment/       # CLI experiment runner
```

## LLM decisions

Set env vars to enable LLM-based country decisions (otherwise heuristic):

```
OPENROUTER_API_KEY=...
ECON_LLM_MODEL=minimax/minimax-m2.5  # optional
```
