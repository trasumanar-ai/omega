"""Chat platform — simüle mesajlaşma servisi.

Discit'in SimChatTransport'u şu endpointleri çağırır:
  POST /chat/send          → mesaj gönder
  POST /chat/mark-read     → okundu işaretle
  GET  /chat/media/{id}/url   → media URL
  GET  /chat/media/{id}/bytes → media indir

CustomerAgent inbound akışı:
  POST /chat/inbound       → müşteri mesajını integrations'a forward et
"""
import logging
import time
import uuid
from datetime import datetime, timezone

import httpx
from fastapi import APIRouter, Request
from fastapi.responses import Response
from pydantic import BaseModel

from omega_chat.hub import broadcast

logger = logging.getLogger("chat.router")

router = APIRouter()

# In-memory mesaj deposu
_messages: list[dict] = []


class InboundRequest(BaseModel):
    phone: str
    message: str
    sender_name: str = ""


class SendRequest(BaseModel):
    type: str = "text"
    to: str
    body: str | None = None
    url: str | None = None
    caption: str | None = None
    filename: str | None = None
    template_name: str | None = None
    language: str | None = None
    parameters: list[dict] | None = None
    interactive_data: dict | None = None
    latitude: float | None = None
    longitude: float | None = None
    name: str | None = None
    address: str | None = None


class MarkReadRequest(BaseModel):
    message_id: str


@router.post("/inbound")
async def inbound(req: InboundRequest, request: Request):
    """CustomerAgent'tan gelen mesajı integrations'a webhook olarak forward et."""
    msg_id = f"sim_{uuid.uuid4().hex[:12]}"
    webhook_payload = {
        "event": "message",
        "data": {
            "id": msg_id,
            "from": req.phone,
            "timestamp": int(time.time()),
            "type": "text",
            "body": req.message,
            "sender_name": req.sender_name,
        },
    }

    # Müşteri cevap verdi → önceki outbound mesajlar "okundu" sayılır
    for m in _messages:
        if m["chat_id"] == req.phone and m["direction"] == "outbound" and m["status"] != "read":
            m["status"] = "read"

    # Lokal kayıt
    msg = {
        "id": msg_id,
        "channel": "whatsapp",
        "chat_id": req.phone,
        "direction": "inbound",
        "type": "text",
        "timestamp": datetime.now(timezone.utc).isoformat(),
        "body": req.message,
        "status": "received",
    }
    _messages.append(msg)
    await broadcast(msg)

    # Integrations'a forward
    integrations_url = getattr(request.app.state, "integrations_url", None)
    if integrations_url:
        try:
            async with httpx.AsyncClient() as client:
                resp = await client.post(
                    f"{integrations_url}/whatsapp/webhook",
                    json=webhook_payload,
                    timeout=30.0,
                )
            logger.info(
                "Inbound forwarded to integrations: %s → %d",
                req.phone, resp.status_code,
            )
        except Exception as e:
            logger.error("Integrations forward failed: %s", e)

    return {"ok": True, "id": msg_id}


@router.post("/send")
async def send(req: SendRequest, request: Request):
    msg = {
        "id": f"sim_{uuid.uuid4().hex[:12]}",
        "channel": "whatsapp",
        "chat_id": req.to,
        "direction": "outbound",
        "type": req.type,
        "timestamp": datetime.now(timezone.utc).isoformat(),
        "body": req.body or req.caption or req.template_name,
        "status": "sent",
    }
    _messages.append(msg)
    await broadcast(msg)

    # Mesajı müşteri agent'ına ilet
    engine = getattr(request.app.state, "sim_engine", None)
    if engine and msg["body"]:
        engine.deliver_to_customer(req.to, msg["body"])

    return msg


@router.post("/mark-read")
async def mark_read(req: MarkReadRequest):
    return {"ok": True}


@router.get("/media/{media_id}/url")
async def media_url(media_id: str):
    return {"url": f"https://sim.local/media/{media_id}"}


@router.get("/media/{media_id}/bytes")
async def media_bytes(media_id: str):
    return Response(content=b"", media_type="application/octet-stream")


@router.get("/messages")
async def list_messages():
    """Debug — tüm mesajları listele."""
    return _messages
