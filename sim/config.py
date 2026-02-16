"""Simülasyon konfigürasyonu — env vars ile kontrol edilir."""

from pydantic_settings import BaseSettings


class SimConfig(BaseSettings):
    model_config = {"env_prefix": "SIM_"}

    # Discit demand webhook hedefi
    demand_url: str = "http://discit-demand-app:13000"

    # Discit integrations API
    integrations_url: str = "http://discit-integrations-api:11000"

    # Zaman çarpanı: 60 = 1 sim-dakika = 1 gerçek saniye
    speed: float = 60.0

    # Kaç sim-dakikada bir lead üret
    lead_interval_minutes: float = 30.0

    # Eşzamanlı müşteri limiti
    max_customers: int = 10

    # LLM ayarları (bp-agent)
    llm_provider: str = "gemini"
    llm_model: str = "gemini-3-flash-preview"
