"""SimEngine — lead üretimi, müşteri yaşam döngüsü, sim time."""

import asyncio
import logging
import random
import uuid

import httpx
from bp_agent import AgentConfig

from sim.clock import SimClock
from sim.config import SimConfig
from sim.customer import CustomerAgent
from sim.personas import PERSONAS, Persona

logger = logging.getLogger("sim.engine")


class SimEngine:
    def __init__(self, config: SimConfig | None = None):
        self.config = config or SimConfig()
        self.clock = SimClock(speed=self.config.speed)
        self._customers: dict[str, CustomerAgent] = {}  # phone → agent
        self._tasks: list[asyncio.Task] = []
        self._lead_gen_task: asyncio.Task | None = None
        self._persona_index = 0

    async def start(self) -> None:
        """Lifespan'da çağrılır — lead gen loop'u başlat."""
        logger.info(
            "SimEngine başlıyor | speed=%.0f | interval=%.0f min | max=%d",
            self.config.speed,
            self.config.lead_interval_minutes,
            self.config.max_customers,
        )
        self._lead_gen_task = asyncio.create_task(self._lead_gen_loop())

    async def stop(self) -> None:
        """Tüm task'ları durdur."""
        logger.info("SimEngine durduruluyor...")
        if self._lead_gen_task:
            self._lead_gen_task.cancel()
        for task in self._tasks:
            task.cancel()
        # Hepsinin bitmesini bekle
        all_tasks = ([self._lead_gen_task] if self._lead_gen_task else []) + self._tasks
        if all_tasks:
            await asyncio.gather(*all_tasks, return_exceptions=True)
        logger.info("SimEngine durdu.")

    async def _lead_gen_loop(self) -> None:
        """Periyodik lead üretimi."""
        # İlk lead'i hemen üret (kısa gecikme)
        await asyncio.sleep(2)
        try:
            while True:
                if len(self._customers) < self.config.max_customers:
                    await self._spawn_lead()
                else:
                    logger.info("Müşteri limiti doldu (%d), bekleniyor...", self.config.max_customers)
                    self._cleanup_done_customers()
                await self.clock.sleep_sim_minutes(self.config.lead_interval_minutes)
        except asyncio.CancelledError:
            logger.info("Lead gen loop iptal edildi.")

    async def _spawn_lead(self) -> None:
        """Yeni bir persona seç, webhook at, CustomerAgent başlat."""
        persona = self._next_persona()

        # Zaten aktif mi?
        if persona.phone in self._customers:
            logger.info("[%s] zaten aktif, atlanıyor.", persona.name)
            return

        # Lead webhook'u discit'e gönder
        lead_payload = {
            "event": "lead",
            "data": {
                "leadgen_id": f"sim_{uuid.uuid4().hex[:8]}",
                "form_id": "sim_form_001",
                "full_name": persona.name,
                "phone_number": persona.phone,
                "email": f"{persona.name.lower().replace(' ', '.')}@sim.local",
            },
        }
        try:
            async with httpx.AsyncClient() as client:
                resp = await client.post(
                    f"{self.config.demand_url}/webhooks/meta/lead",
                    json=lead_payload,
                    timeout=10.0,
                )
                logger.info(
                    "Lead webhook gönderildi: %s → %s",
                    persona.name, resp.json(),
                )
        except Exception as e:
            logger.error("Lead webhook gönderilemedi: %s — %s", persona.name, e)
            return

        # Kısa gecikme — müşterinin reklamı görüp yazmasını simüle et
        await self.clock.sleep_sim_minutes(random.uniform(1, 5))

        # CustomerAgent başlat
        agent_config = AgentConfig(
            provider=self.config.llm_provider,
            model=self.config.llm_model,
            enable_builtin_tools=False,
            max_iterations=25,
        )
        customer = CustomerAgent(
            persona=persona,
            clock=self.clock,
            demand_url=self.config.demand_url,
            config=agent_config,
        )
        self._customers[persona.phone] = customer
        task = asyncio.create_task(customer.run())
        self._tasks.append(task)
        logger.info("[%s] müşteri agent başlatıldı.", persona.name)

    def deliver_to_customer(self, phone: str, text: str) -> bool:
        """Chat router'dan gelen mesajı müşteriye ilet."""
        customer = self._customers.get(phone)
        if customer and not customer.is_done:
            customer.receive_message(text)
            return True
        logger.warning("Müşteri bulunamadı veya konuşma bitmiş: %s", phone)
        return False

    def _next_persona(self) -> Persona:
        """Round-robin persona seçimi."""
        persona = PERSONAS[self._persona_index % len(PERSONAS)]
        self._persona_index += 1
        return persona

    def _cleanup_done_customers(self) -> None:
        """Bitmiş müşterileri temizle."""
        done_phones = [
            phone for phone, c in self._customers.items() if c.is_done
        ]
        for phone in done_phones:
            del self._customers[phone]
            logger.info("Müşteri temizlendi: %s", phone)
        # Bitmiş task'ları da temizle
        self._tasks = [t for t in self._tasks if not t.done()]
