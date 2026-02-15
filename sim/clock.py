"""Simülasyon saati — speed multiplier ile gerçek zamanı sim zamanına çevirir."""

import asyncio
import time


class SimClock:
    def __init__(self, speed: float = 60.0):
        # speed=60 → 1 sim-dakika = 1 gerçek saniye
        self.speed = speed

    def to_real_seconds(self, sim_seconds: float) -> float:
        """Sim saniyelerini gerçek saniyelere çevir."""
        return sim_seconds / self.speed

    def sleep_sim_sync(self, sim_seconds: float) -> None:
        """Senkron (thread içi) — sim_seconds kadar sim zamanı bekle."""
        time.sleep(self.to_real_seconds(sim_seconds))

    async def sleep_sim_minutes(self, minutes: float) -> None:
        """Async — minutes kadar sim dakikası bekle."""
        real_seconds = self.to_real_seconds(minutes * 60)
        await asyncio.sleep(real_seconds)
