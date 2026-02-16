"""SimEngine — müşteri üretimi, yaşam döngüsü, sim time.

Discit demand /message endpoint'i lead yoksa otomatik yaratır,
dolayısıyla ayrı lead webhook'a gerek yok — ilk mesaj yeterli.
"""

import asyncio
import logging
import random

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
        self._spawn_task: asyncio.Task | None = None
        self._persona_index = 0

    async def start(self) -> None:
        """Lifespan'da çağrılır — müşteri üretim loop'u başlat."""
        logger.info(
            "SimEngine başlıyor | speed=%.0f | interval=%.0f min | max=%d",
            self.config.speed,
            self.config.lead_interval_minutes,
            self.config.max_customers,
        )
        self._spawn_task = asyncio.create_task(self._spawn_loop())

    async def stop(self) -> None:
        """Tüm task'ları durdur."""
        logger.info("SimEngine durduruluyor...")
        if self._spawn_task:
            self._spawn_task.cancel()
        for task in self._tasks:
            task.cancel()
        all_tasks = ([self._spawn_task] if self._spawn_task else []) + self._tasks
        if all_tasks:
            await asyncio.gather(*all_tasks, return_exceptions=True)
        logger.info("SimEngine durdu.")

    async def _spawn_loop(self) -> None:
        """Periyodik müşteri üretimi."""
        await asyncio.sleep(2)
        try:
            while True:
                self._cleanup_done_customers()
                if len(self._customers) < self.config.max_customers:
                    await self._spawn_customer()
                else:
                    logger.info("Müşteri limiti doldu (%d), bekleniyor...", self.config.max_customers)
                await self.clock.sleep_sim_minutes(self.config.lead_interval_minutes)
        except asyncio.CancelledError:
            logger.info("Spawn loop iptal edildi.")

    async def _spawn_customer(self) -> None:
        """Yeni bir persona seç, CustomerAgent başlat.

        İlk mesaj /message'a gidince discit otomatik lead oluşturur.
        """
        persona = self._next_persona()

        if persona.phone in self._customers:
            logger.info("[%s] zaten aktif, atlanıyor.", persona.name)
            return

        agent_config = AgentConfig(
            provider=self.config.llm_provider,
            model=self.config.llm_model,
            enable_builtin_tools=False,
            max_iterations=25,
        )
        customer = CustomerAgent(
            persona=persona,
            clock=self.clock,
            platform_url="http://localhost:20000",
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
