# Changelog

Format: [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), SemVer.

## [Unreleased]

### Added
- Country-level economic simulation engine (`go-sim/internal/econ`).
  4 countries with energy generation, resource extraction, quote-based trading,
  and 3 market algorithms (`no_trade`, `linear`, `scarcity_spike`).
- HTTP API server (`go-sim/cmd/omega-econ-server`) with state, step, reset,
  config, runs, compare, lab, and experiment endpoints.
- CLI experiment runner (`go-sim/cmd/omega-econ`) with hypothesis testing,
  control/variant scenarios, and JSON result output.
- Lab session system for A/B testing preset x algorithm combinations.
- React frontend (`src/EconApp.tsx`) with overview map, market details,
  country panels, lab management, and run history.
- OpenRouter LLM integration for country build decisions.

### Removed
- Old grid-world agent simulation (fruit/vitamin/genome mechanics).
- Old Go simulation server and CLI.
- Stale Docker/nginx config (frontend-only, didn't include Go backend).
