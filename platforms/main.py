"""Omega platforms — simüle dış dünya servisleri + aktif simülasyon engine."""

import logging
from contextlib import asynccontextmanager

from fastapi import FastAPI

from chat.router import router as chat_router
from ads.router import router as ads_router
from ecommerce.router import router as ecommerce_router
from sim.engine import SimEngine

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(name)s] %(levelname)s: %(message)s",
)


@asynccontextmanager
async def lifespan(app: FastAPI):
    engine = SimEngine()
    app.state.sim_engine = engine
    await engine.start()
    yield
    await engine.stop()


app = FastAPI(title="Omega Platforms", version="0.2.0", lifespan=lifespan)

app.include_router(chat_router, prefix="/chat", tags=["chat"])
app.include_router(ads_router, prefix="/ads", tags=["ads"])
app.include_router(ecommerce_router, prefix="/ecommerce", tags=["ecommerce"])


@app.get("/health")
async def health():
    return {"status": "ok"}
