# Omega

Simulation environment for testing [discit](https://github.com/tunagul/discit).

Fake platforms, real conversations, autonomous AI customers.

> For the developer's vision and thoughts behind this project, see [HUMANS.md](HUMANS.md).

```
PLATFORMS  (20000)  simulated external services (chat, ads, ecommerce)
SIM ENGINE          autonomous customer agents, time simulation
```

## Architecture

```
┌─────────────────────────────────────────────┐
│  Omega Platforms (:20000)                   │
│                                             │
│  ┌──────────┐ ┌──────────┐ ┌────────────┐  │
│  │   Chat   │ │   Ads    │ │ Ecommerce  │  │
│  │  /chat/* │ │  /ads/*  │ │/ecommerce/*│  │
│  └──────────┘ └──────────┘ └────────────┘  │
│                                             │
│  ┌──────────────────────────────────────┐   │
│  │  SimEngine                          │   │
│  │  ├─ SimClock (speed=60)             │   │
│  │  ├─ CustomerAgent (bp-agent) x N    │   │
│  │  └─ Persona pool (10 people)        │   │
│  └──────────────────────────────────────┘   │
└──────────────────┬──────────────────────────┘
                   │ HTTP
                   ▼
┌──────────────────────────────────────────────┐
│  Discit                                      │
│  ├─ demand-app     (:13000)  ← messages      │
│  ├─ supply-tools   (:8011)                   │
│  └─ postgres       (:5432)                   │
└──────────────────────────────────────────────┘
```

## How it works

Discit is a reactive system -- nothing happens until a customer message or lead arrives. Omega fills that gap.

1. SimEngine starts and spawns customers at regular intervals
2. Each CustomerAgent sends its first message → `POST /message` → discit demand
3. Discit receives the message, creates a lead, wakes its sales agent, generates a reply
4. Discit's reply → `POST /chat/send` → omega chat router
5. Chat router delivers the message to the CustomerAgent's inbox
6. CustomerAgent reads inbox, thinks, replies or ends the conversation

The customers are LLM-powered agents with distinct personalities. They behave like real people: they ask questions, get impatient, compare prices, or ghost you. See [AGENTS.md](AGENTS.md) for details.

## Quick start

```bash
cp .env.example .env
# Fill in at least GEMINI_API_KEY

# Bring everything up (discit + omega)
make up

# Watch the logs
make logs

# Health check
make health

# Tear down
make down
```

Services will be available at:

| Service | Port | Description |
|---------|------|-------------|
| omega-platforms | 20000 | Simulated platforms + sim engine |
| discit-demand-app | 13000 | Discit sales + marketing |
| discit-supply-tools | 8011 | Discit business logic |

## Module overview

### Platforms (`platforms/`)

Stub API servers that replace real external services. Discit talks to these the same way it would talk to WhatsApp, Meta Ads, or Shopify.

- **Chat** (`/chat/*`): Message sending, read receipts, media URLs. When discit sends a message, the chat router forwards it to the corresponding CustomerAgent.
- **Ads** (`/ads/*`): Campaign CRUD, ad sets, insights, conversion events. Returns plausible empty/stub data.
- **Ecommerce** (`/ecommerce/*`): Order creation and retrieval, customer lookup. In-memory store.

All three run as a single FastAPI service.

### Sim (`sim/`)

The active simulation engine. This is what makes omega more than just stub APIs.

- **SimEngine**: Orchestrator. Spawns customers on a timer, manages their lifecycle, cleans up when they're done.
- **CustomerAgent**: Autonomous LLM agent (built on [bp-agent](https://github.com/tunagul/blueprint)) that acts as a real customer. Has tools for sending messages, checking inbox, waiting, and ending the conversation.
- **SimClock**: Time multiplier. `speed=60` means 1 sim-minute = 1 real second. A 30-minute customer patience window plays out in 30 seconds.
- **Personas**: 10 Turkish customer templates with different personalities and patience levels.

## Configuration

All sim settings are controlled via `SIM_`-prefixed environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `SIM_DEMAND_URL` | `http://discit-demand-app:13000` | Discit demand endpoint |
| `SIM_SPEED` | `60` | Time multiplier (60 = 1 sim-minute per real second) |
| `SIM_LEAD_INTERVAL_MINUTES` | `30` | Sim-minutes between new customers |
| `SIM_MAX_CUSTOMERS` | `10` | Max concurrent customers |
| `SIM_LLM_PROVIDER` | `gemini` | LLM provider for customer agents |
| `SIM_LLM_MODEL` | `gemini-3-flash-preview` | LLM model for customer agents |

Gemini key rotation is supported: `GEMINI_API_KEY` through `GEMINI_API_KEY_5`.

## Project structure

```
omega/
├── platforms/              # FastAPI service -- simulated external world
│   ├── main.py             # App entrypoint + lifespan (starts sim engine)
│   ├── chat/router.py      # WhatsApp-like messaging API
│   ├── ads/router.py       # Meta Ads-like advertising API
│   ├── ecommerce/router.py # Shopify-like ecommerce API
│   ├── Dockerfile
│   └── requirements.txt
├── sim/                    # Simulation engine
│   ├── engine.py           # Customer spawning + lifecycle management
│   ├── customer.py         # Autonomous customer agent (bp-agent)
│   ├── clock.py            # Time multiplier
│   ├── personas.py         # 10 Turkish customer personas
│   └── config.py           # Env-based configuration
├── docker-compose.yml
├── Makefile
└── BLUEPRINT.yaml
```

## Blueprints

Every directory has a `BLUEPRINT.yaml` describing its intent and structure. These serve as design documents for AI-assisted development. See [blueprint](https://github.com/tunagul/blueprint) for the tooling.

## License

[GPLv3](LICENSE) -- Copyright 2026 Tuna Gul
