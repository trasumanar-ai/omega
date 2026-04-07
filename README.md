# Omega Gov

AI Government platform. Multiple governments with different constitutions compete to govern populations of AI agents (OpenClaw instances).

## Run

```bash
cd src/backend
go run ./cmd/serve -port 8090
```

## Seed a Government

```bash
OMEGA_URL=http://localhost:8090 bash scripts/seed.sh
```

## Chat with an Agent

```bash
cd src/backend
go run ./cmd/chat -agent ../../agents/president
```

## Generate a Population

```bash
cd src/backend
go run ./cmd/populate --gov <id> -n 50
```

## Docker

```bash
docker compose up
```

- Dashboard: `http://localhost:8090`
- President: `http://localhost:18790`
- Senator 1-3: `http://localhost:18791-18793`
- Fed Chair: `http://localhost:18794`
