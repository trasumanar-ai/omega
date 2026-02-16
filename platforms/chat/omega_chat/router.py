"""Omega Chat Viewer — API + WebSocket + static file serving."""

import logging
from collections import defaultdict
from pathlib import Path

from fastapi import APIRouter, WebSocket, WebSocketDisconnect
from fastapi.responses import FileResponse

from chat.omega_chat.hub import connect, disconnect

logger = logging.getLogger("omega-chat.router")

router = APIRouter()

_static_dir = Path(__file__).parent / "static"


@router.get("/")
async def index():
    return FileResponse(_static_dir / "index.html", media_type="text/html")


@router.get("/api/conversations")
async def conversations():
    """Tüm konuşmaları listele — chat_id'ye göre grupla, son mesaj + sayı."""
    from chat.router import _messages

    groups: dict[str, list[dict]] = defaultdict(list)
    for msg in _messages:
        groups[msg["chat_id"]].append(msg)

    contacts = _get_contacts()
    result = []
    for chat_id, msgs in groups.items():
        last = msgs[-1]
        result.append({
            "chat_id": chat_id,
            "name": contacts.get(chat_id, chat_id),
            "last_message": last.get("body", ""),
            "last_timestamp": last.get("timestamp", ""),
            "count": len(msgs),
        })

    result.sort(key=lambda c: c["last_timestamp"], reverse=True)
    return result


@router.get("/api/conversations/{chat_id}")
async def conversation_messages(chat_id: str):
    """Bir konuşmanın tüm mesajlarını döndür."""
    from chat.router import _messages

    return [m for m in _messages if m["chat_id"] == chat_id]


@router.get("/api/contacts")
async def contacts():
    """Persona phone→name mapping."""
    return _get_contacts()


@router.websocket("/ws")
async def websocket_endpoint(ws: WebSocket):
    await connect(ws)
    try:
        while True:
            await ws.receive_text()  # keep alive, read-only
    except WebSocketDisconnect:
        pass
    finally:
        await disconnect(ws)


@router.get("/api/profiles/{phone}")
async def profile(phone: str):
    """Müşteri profil bilgisi — persona + system prompt + LLM config."""
    persona = _get_persona(phone)
    if not persona:
        from fastapi.responses import JSONResponse
        return JSONResponse({"error": "not found"}, status_code=404)

    try:
        from sim.customer import SYSTEM_PROMPT_TEMPLATE
        system_prompt = SYSTEM_PROMPT_TEMPLATE.format(
            name=persona.name,
            personality=persona.personality,
        )
    except ImportError:
        system_prompt = ""

    try:
        from sim.config import SimConfig
        cfg = SimConfig()
        llm = {"provider": cfg.llm_provider, "model": cfg.llm_model}
    except ImportError:
        llm = {"provider": "unknown", "model": "unknown"}

    return {
        "name": persona.name,
        "phone": persona.phone,
        "personality": persona.personality,
        "patience_minutes": persona.patience_minutes,
        "system_prompt": system_prompt,
        "llm": llm,
        "framework": "bp-agent v0.3.0",
        "tools": ["send_message", "check_inbox", "wait", "done"],
    }


def _get_persona(phone: str):
    try:
        from sim.personas import PERSONAS
        for p in PERSONAS:
            if p.phone == phone:
                return p
    except ImportError:
        pass
    return None


def _get_contacts() -> dict[str, str]:
    try:
        from sim.personas import PERSONAS
        return {p.phone: p.name for p in PERSONAS}
    except ImportError:
        return {}
