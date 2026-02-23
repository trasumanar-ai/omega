# Omega Go Simulation Core

This directory contains a standalone Go port of the internal simulation logic.

## Run

```bash
cd go-sim
go run ./cmd/omega-sim -steps 1000 -seed 42
```

Optional flags:

- `-width`, `-height`
- `-agents`, `-slots`
- `-sample-every`
- `-out` (write JSON report to file)

Example with output file:

```bash
go run ./cmd/omega-sim -steps 5000 -agents 300 -out run.json
```

## Run HTTP Server (for Web UI)

```bash
cd go-sim
go run ./cmd/omega-sim-server -port 8080
```

Optional flags:

- `-runs-dir` (default `./runs`, persists web-server run logs)

Web UI (`vite`) calls this API:

- `GET /api/sim/state`
- `POST /api/sim/step`
- `POST /api/sim/reset`
- `POST /api/sim/config`
- `POST /api/sim/replay` (seed + config ile deterministik yeniden kurulum)
- `GET /api/sim/agent/:id`
- `GET /api/version`
- `GET /api/sim/runs`
- `GET /api/sim/runs/:id`

Then run frontend (project root):

```bash
cd ..
npm run dev
```

Each run is saved under `go-sim/runs`:

- `run_index.json` (recent run summaries)
- `run_<id>.json` (full run details + trace)
- `active_run.json` (in-progress run)

Run dosyalarinda `schemaVersion`, `serverVersion`, `apiVersion` alanlari bulunur.
