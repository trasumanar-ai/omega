# Omega

Repo is under active rewrite. Architecture, API shape, and product direction are intentionally unstable.

## Run

```bash
npm install
npm run server
npm run dev
```

- UI: `http://localhost:20000`
- API: `http://localhost:8090`

## Optional LLM

```bash
export OPENROUTER_API_KEY=...
export ECON_LLM_MODEL=deepseek/deepseek-chat-v3-0324
```

## Current Rule

Optimize for iteration speed. Treat in-repo docs as provisional unless they are needed to run the code.

## License

[GPLv3](LICENSE)
