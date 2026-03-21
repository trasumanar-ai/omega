# Omega

Repo is under active rewrite. Architecture, API shape, and product direction are intentionally unstable.

## Current Focus

Simulation and CLI work. UI is parked under `src/frontend` and is not the primary surface right now.

## Backend Run

```bash
cd src/backend
go run ./cmd/serve -port 8090
```

## Optional LLM

```bash
export OPENROUTER_API_KEY=...
export ECON_LLM_MODEL=deepseek/deepseek-chat-v3-0324
```

## Frontend

```bash
cd src/frontend
npm install
npm run dev
```

- UI: `http://localhost:20000`
- API proxy target: `http://localhost:8090`

## Current Rule

Optimize for iteration speed. Treat in-repo docs as provisional unless they are needed to run the code.

## License

[GPLv3](LICENSE)
