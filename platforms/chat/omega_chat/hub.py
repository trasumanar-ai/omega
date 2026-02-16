"""WebSocket broadcast hub — bağlı tüm viewer client'lara mesaj yayınlar."""

import json
import logging

from fastapi import WebSocket

logger = logging.getLogger("omega-chat.hub")

_clients: set[WebSocket] = set()


async def connect(ws: WebSocket):
    await ws.accept()
    _clients.add(ws)
    logger.info("Viewer connected (%d total)", len(_clients))


async def disconnect(ws: WebSocket):
    _clients.discard(ws)
    logger.info("Viewer disconnected (%d total)", len(_clients))


async def broadcast(message: dict):
    """Mesajı tüm bağlı client'lara JSON olarak gönder."""
    if not _clients:
        return
    payload = json.dumps(message, ensure_ascii=False)
    dead: list[WebSocket] = []
    for ws in _clients:
        try:
            await ws.send_text(payload)
        except Exception:
            dead.append(ws)
    for ws in dead:
        _clients.discard(ws)
