"""Chat platform — simüle mesajlaşma servisi.

Discit'in SimChatTransport'u şu endpointleri çağırır:
  POST /chat/send          → mesaj gönder
  POST /chat/mark-read     → okundu işaretle
  GET  /chat/media/{id}/url   → media URL
  GET  /chat/media/{id}/bytes → media indir
"""
import uuid
from datetime import datetime, timezone

from fastapi import APIRouter
from fastapi.responses import Response
from pydantic import BaseModel

router = APIRouter()

# In-memory mesaj deposu
_messages: list[dict] = []


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


@router.post("/send")
async def send(req: SendRequest):
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
