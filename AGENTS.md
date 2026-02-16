# Agents

This document describes the AI agents that live inside omega.

## CustomerAgent

**File:** `sim/customer.py`
**Framework:** [bp-agent](https://github.com/tunagul/blueprint) v0.3.0
**Default LLM:** Gemini 3 Flash

An autonomous agent that pretends to be a real customer. Each one runs in its own thread, has a personality, and carries on a WhatsApp-style conversation with discit's sales agent.

### Lifecycle

1. SimEngine picks a persona and creates a CustomerAgent
2. Agent receives its system prompt -- persona details, rules, available tools
3. Agent sends its opening message ("saw your ad, interested in...")
4. Loop: wait → check inbox → think → reply or hang up
5. If patience runs out, the agent either sends a follow-up or ghosts

### Tools

The agent has no access to bp-agent's built-in tools. Only these four:

| Tool | What it does |
|------|-------------|
| `send_message(text)` | Posts a message to discit demand (`POST /message`) |
| `check_inbox()` | Reads messages from a thread-safe queue |
| `wait(minutes)` | Sleeps for the given sim-minutes |
| `done()` | Ends the conversation |

### Personas

10 Turkish customer personas defined in `sim/personas.py`. Each has a name, phone number, personality description, and patience level (in sim-minutes).

| Name | Personality | Patience |
|------|------------|----------|
| Ayse Yilmaz | Impatient, asks for price immediately | 10 min |
| Mehmet Kaya | Indecisive, asks lots of questions | 30 min |
| Fatma Demir | Polite but distant, formal language | 20 min |
| Ali Celik | Direct buyer, few words | 15 min |
| Zeynep Aksoy | Emoji heavy, fast typer, young | 12 min |
| Hasan Ozturk | Skeptical, needs convincing | 25 min |
| Elif Sahin | Compares prices, haggles | 20 min |
| Burak Aydin | Technical questions, wants specs | 30 min |
| Selin Koc | Friendly and chatty, slow to decide | 35 min |
| Emre Yildiz | Busy professional, wants it short | 8 min |

### Constraints

- System prompt bans words like "AI", "simulation", "agent"
- Turkish only, casual WhatsApp language
- Max 20 turns per conversation
- One active conversation per persona

## SimEngine

**File:** `sim/engine.py`

The orchestrator. Doesn't generate text itself -- it manages the CustomerAgents.

- Spawns new customers at configurable intervals (round-robin persona selection)
- Enforces the max concurrent customer limit
- Cleans up finished conversations
- Routes incoming messages from discit to the right customer's inbox

## SimClock

**File:** `sim/clock.py`

Converts real time to sim time using a speed multiplier.

```
speed = 60  →  1 sim-minute = 1 real second
                30 sim-minutes = 30 real seconds
                8 sim-hours ≈ 8 real minutes
```

Provides both sync (`time.sleep`, for agent threads) and async (`asyncio.sleep`, for the engine loop) sleep methods.

## Discit's agents (external)

The agents on discit's side are not controlled by omega, but they're the other half of every conversation:

- **Sales agents** (demand): Handle customer conversations, manage pipeline stages, create payment links
- **Cooper** (supply): Operations agent for inventory and production queries

These agents read from and write to omega's platform stubs. The whole point of omega is to give them something to react to.
