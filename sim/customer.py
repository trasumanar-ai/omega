"""CustomerAgent — bp-agent tabanlı otonom müşteri simülatörü.

Her müşteri kendi thread'inde çalışır, WhatsApp tarzı mesajlaşır.
"""

import asyncio
import logging
import queue

import httpx
from bp_agent import Agent, AgentConfig
from bp_agent.tools import build_schema

from sim.clock import SimClock
from sim.personas import Persona

logger = logging.getLogger("sim.customer")

SYSTEM_PROMPT_TEMPLATE = """\
Sen gerçek bir müşteri olan {name} rolündesin. Bir WhatsApp konuşmasındasın.
Bir işletmeyle ilk kez iletişime geçiyorsun — ya bir reklam gördün ya da sana ulaşıldı.

KİŞİLİĞİN:
{personality}

KURALLAR:
- Türkçe yaz, doğal WhatsApp tarzında (kısa mesajlar, günlük dil).
- Gerçek bir insan gibi davran — robot gibi yazma.
- Her mesajda tek bir şey sor veya söyle, uzun paragraflar yazma.
- İlk mesajında kendini tanıtma — direkt konuya gir (reklam gördüm, bilgi almak istiyorum, vs.)
- Cevap gelmezse sabırsızlan veya konuşmayı bırak.
- İlgini çekerse satın alma yönünde ilerle, çekmezse nazikçe bitir.
- ASLA "yapay zeka", "simülasyon", "agent" gibi kelimeler kullanma.

ARAÇLARIN:
- send_message: İşletmeye mesaj göndermek için kullan.
- check_inbox: Gelen mesajları kontrol etmek için kullan.
- wait: Cevap beklemek için kullan (dakika cinsinden sim zamanı).
- done: Konuşmayı bitirmek istediğinde çağır.

AKIŞ:
1. İlk mesajını at (send_message ile).
2. Bir süre bekle (wait ile, 2-5 dakika).
3. Gelen kutuyu kontrol et (check_inbox ile).
4. Mesaj varsa oku, düşün, cevapla.
5. Mesaj yoksa biraz daha bekle.
6. Sabrın tükendiyse ya takip mesajı at ya da done ile bitir.
7. Konuşma doğal bir sonuca ulaştıysa done ile bitir.
"""


class CustomerAgent:
    def __init__(
        self,
        persona: Persona,
        clock: SimClock,
        demand_url: str,
        config: AgentConfig,
    ):
        self.persona = persona
        self.clock = clock
        self.demand_url = demand_url
        self._inbox: queue.Queue[str] = queue.Queue()
        self._done = False
        self._task: asyncio.Task | None = None

        # bp-agent oluştur
        self._agent = Agent(
            name=f"customer-{persona.phone}",
            config=config,
            system_prompt=SYSTEM_PROMPT_TEMPLATE.format(
                name=persona.name,
                personality=persona.personality,
            ),
        )

        # Custom tool'ları ekle
        self._register_tools()

    def _register_tools(self) -> None:
        """Müşteri agent'ına özel araçları kaydet."""
        agent = self._agent

        # --- send_message ---
        def send_message(text: str) -> str:
            return self._tool_send_message(text)

        agent.add_tool(
            "send_message",
            send_message,
            build_schema(
                "send_message",
                "İşletmeye WhatsApp mesajı gönder.",
                text={"type": "string", "description": "Gönderilecek mesaj metni", "required": True},
            ),
        )

        # --- check_inbox ---
        def check_inbox() -> str:
            return self._tool_check_inbox()

        agent.add_tool(
            "check_inbox",
            check_inbox,
            build_schema(
                "check_inbox",
                "Gelen kutuyu kontrol et. İşletmeden gelen mesajları oku.",
            ),
        )

        # --- wait ---
        def wait(minutes: int = 3) -> str:
            return self._tool_wait(minutes)

        agent.add_tool(
            "wait",
            wait,
            build_schema(
                "wait",
                "Belirtilen süre kadar bekle (sim dakikası).",
                minutes={"type": "integer", "description": "Beklenecek dakika (sim zamanı)", "required": True},
            ),
        )

        # --- done ---
        def done() -> str:
            return self._tool_done()

        agent.add_tool(
            "done",
            done,
            build_schema(
                "done",
                "Konuşmayı bitir ve ayrıl.",
            ),
        )

    # ── Tool implementations ──

    def _tool_send_message(self, text: str) -> str:
        """Mesajı discit demand webhook'una gönder."""
        payload = {
            "event": "message",
            "data": {
                "chat_id": self.persona.phone,
                "body": text,
                "sender_name": self.persona.name,
            },
        }
        try:
            resp = httpx.post(
                f"{self.demand_url}/webhooks/whatsapp/message",
                json=payload,
                timeout=30.0,
            )
            data = resp.json()
            logger.info(
                "[%s] → mesaj gönderildi: %s | cevap: %s",
                self.persona.name, text[:50], data,
            )
            # Discit'in senkron cevapları varsa inbox'a koy
            responses = data.get("responses", [])
            for r in responses:
                self._inbox.put(r)
            return f"Mesaj gönderildi. {'Cevap geldi: ' + '; '.join(responses) if responses else 'Henüz cevap yok.'}"
        except Exception as e:
            logger.error("[%s] mesaj gönderilemedi: %s", self.persona.name, e)
            return f"Hata: mesaj gönderilemedi ({e})"

    def _tool_check_inbox(self) -> str:
        """Inbox'taki mesajları oku."""
        messages = []
        while not self._inbox.empty():
            try:
                messages.append(self._inbox.get_nowait())
            except queue.Empty:
                break
        if messages:
            return "Gelen mesajlar:\n" + "\n".join(f"- {m}" for m in messages)
        return "Gelen kutusu boş — henüz mesaj yok."

    def _tool_wait(self, minutes: int = 3) -> str:
        """Sim zamanında bekle."""
        self.clock.sleep_sim_sync(minutes * 60)
        return f"{minutes} dakika beklendi."

    def _tool_done(self) -> str:
        """Konuşmayı bitir."""
        self._done = True
        return "Konuşma bitirildi."

    # ── Dış API ──

    def receive_message(self, text: str) -> None:
        """Dışarıdan gelen mesajı inbox'a koy (chat router çağırır)."""
        self._inbox.put(text)
        logger.info("[%s] ← mesaj alındı: %s", self.persona.name, text[:50])

    async def run(self) -> None:
        """Agent döngüsünü asyncio.to_thread ile başlat."""
        self._task = asyncio.current_task()
        try:
            await asyncio.to_thread(self._agent_loop)
        except asyncio.CancelledError:
            logger.info("[%s] iptal edildi", self.persona.name)
        except Exception:
            logger.exception("[%s] agent loop hatası", self.persona.name)

    def _agent_loop(self) -> None:
        """Senkron agent döngüsü — thread'de çalışır."""
        logger.info("[%s] konuşma başlıyor", self.persona.name)

        # İlk mesaj: agent'a konuşmayı başlatmasını söyle
        opening = (
            f"Sen {self.persona.name} adlı bir müştersin. "
            f"Bir işletmenin reklamını gördün ve WhatsApp'tan iletişime geçmek istiyorsun. "
            f"İlk mesajını gönder (send_message kullan). "
            f"Sabır süren {self.persona.patience_minutes} dakika (sim zamanı). "
            f"Bu sürede cevap gelmezse takip mesajı at veya konuşmayı bitir."
        )

        max_turns = 20
        turn = 0
        response = self._agent.chat(opening)
        logger.debug("[%s] agent turn 0: %s", self.persona.name, response[:100] if response else "")

        while not self._done and turn < max_turns:
            turn += 1
            # Agent'a devam etmesini söyle
            prompt = "Devam et. Gelen kutuyu kontrol et, mesaj varsa cevapla, yoksa bekle veya bitir."
            response = self._agent.chat(prompt)
            logger.debug("[%s] agent turn %d: %s", self.persona.name, turn, response[:100] if response else "")

        logger.info("[%s] konuşma bitti (turns=%d, done=%s)", self.persona.name, turn, self._done)

    @property
    def is_done(self) -> bool:
        return self._done
