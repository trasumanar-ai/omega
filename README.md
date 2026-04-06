# Omega

AI Government platform. Multiple governments with different constitutions compete to govern populations of AI agents.

## Run

```bash
cd src/backend
go run ./cmd/serve -port 8090
```

## API

See `src/backend/internal/api/server.go` for all endpoints.

```bash
# Create a government
curl -X POST http://localhost:8090/api/governments \
  -H 'Content-Type: application/json' \
  -d '{"name":"Free Market","config":{"initial_balance":1000,"transfer_tax":0.02,"open_membership":true}}'

# Register an agent
curl -X POST http://localhost:8090/api/governments/{id}/registry/register \
  -H 'Content-Type: application/json' \
  -d '{"name":"Agent Alpha","soul_md":"I am helpful","model":"gpt-4o"}'
```

## License

[GPLv3](LICENSE)
