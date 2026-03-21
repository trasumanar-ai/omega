# Omega

Country-level economic simulation with energy networks, resource trading, and AI-driven decision making.

> For the philosophy behind this project, see [HUMANS.md](HUMANS.md) ([English](HUMANS.en.md)).

## What it does

4 countries on a network extract resources (coal, oil, copper, silicon), generate energy, trade with neighbors, and build infrastructure. Each country makes build decisions via heuristic or LLM. Countries that can't meet energy demand lose stability and eventually die.

3 market algorithms: `no_trade`, `linear`, `scarcity_spike`.

## Quick start

```bash
npm install

# Terminal 1: Go backend
npm run server

# Terminal 2: Frontend
npm run dev
```

Frontend: http://localhost:20000
Backend API: http://localhost:8090

### LLM decisions (optional)

Set `OPENROUTER_API_KEY` env var before starting the backend. Default model: `minimax/minimax-m2.5`.

## Project structure

```
omega/
├── src/
│   ├── web/                      # React frontend (Vite + TypeScript)
│   │   ├── EconApp.tsx           # Main UI
│   │   ├── econ/types.ts         # Type definitions
│   │   ├── hooks/useEconSimulation.ts
│   │   └── ui/charts/            # Recharts wrappers
│   │
│   └── server/                   # Go backend
│       ├── internal/econ/        # Simulation engine
│       ├── cmd/serve/            # HTTP API server
│       └── cmd/experiment/       # CLI experiment runner
│
├── IDEAS.md                      # Future direction: AI Government experiments
└── HUMANS.md                     # Project philosophy
```

## API

All endpoints under `/api/econ/`:

| Method | Path | Description |
|--------|------|-------------|
| GET | `/state` | Current world state |
| POST | `/step` | Advance N ticks |
| POST | `/reset` | Reset simulation |
| POST | `/config` | Update config and reset |
| GET | `/runs` | List past runs |
| GET | `/runs/:id` | Get run details |
| POST | `/compare` | Run algorithm comparison |
| GET | `/lab` | Lab sessions |
| GET | `/experiments` | CLI experiment results |

## License

[GPLv3](LICENSE)
