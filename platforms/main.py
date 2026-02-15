"""Omega platforms — simüle dış dünya servisleri."""
from fastapi import FastAPI

from chat.router import router as chat_router
from ads.router import router as ads_router
from ecommerce.router import router as ecommerce_router

app = FastAPI(title="Omega Platforms", version="0.1.0")

app.include_router(chat_router, prefix="/chat", tags=["chat"])
app.include_router(ads_router, prefix="/ads", tags=["ads"])
app.include_router(ecommerce_router, prefix="/ecommerce", tags=["ecommerce"])


@app.get("/health")
async def health():
    return {"status": "ok"}
