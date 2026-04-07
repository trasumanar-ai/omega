# Population Data

Real-world demographics of deployed OpenClaw agents, collected 2026-04-07.

## Files

| File | What | Source |
|---|---|---|
| `agent-categories.json` | 22 agent types with real frequency | awesome-openclaw-agents (196 files) |
| `model-usage.json` | 9 LLM models with usage weights | OpenRouter rankings + community |
| `personality-traits.json` | 8 comm styles, 22 tone keywords, constraints | Analysis of 196 SOUL.md files |
| `ecosystem.json` | Registries, multi-agent systems, platform stats | Multiple public repos |

## Key Numbers

- **196** production SOUL.md templates analyzed (awesome-openclaw-agents)
- **4,631** total souls in souls.directory (~200-300 human-authored)
- **5,198** curated skills on ClawHub
- **350,000** GitHub stars on OpenClaw
- **41** LLM providers supported
- **348** models available on OpenRouter

## How to Use

```bash
# Generate a realistic population from these distributions
cd src/backend
go run ./cmd/populate --dry-run -n 100       # preview
go run ./cmd/populate --gov <id> -n 50       # register with government
go run ./cmd/populate --gov <id> -n 100 -seed 42  # reproducible
```

## Top 5 Agent Categories

1. Marketing (14.3%)
2. Development (9.2%)
3. Business (7.1%)
4. Creative (6.6%)
5. DevOps / Finance (5.1% each)

## Top 3 Models

1. Claude Sonnet 4.5 (25%)
2. Gemini 2.5 Flash (18%)
3. DeepSeek V3 (15%)

## Dominant Personality Profile

The typical OpenClaw agent is: **concise (57%), analytical (36%), professional (27%), thorough (26%)**. Creative and empathetic traits are rare (7% and 15%).
