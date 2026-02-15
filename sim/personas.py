"""Türkçe müşteri persona şablonları."""

from dataclasses import dataclass


@dataclass
class Persona:
    name: str
    phone: str
    personality: str
    patience_minutes: int  # sim time — cevap bekleyeceği süre


PERSONAS: list[Persona] = [
    Persona(
        name="Ayşe Yılmaz",
        phone="905550001001",
        personality="Sabırsız, kısa ve net mesajlar atar. Hemen fiyat sorar.",
        patience_minutes=10,
    ),
    Persona(
        name="Mehmet Kaya",
        phone="905550001002",
        personality="Kararsız, çok soru sorar. Her detayı öğrenmek ister.",
        patience_minutes=30,
    ),
    Persona(
        name="Fatma Demir",
        phone="905550001003",
        personality="Nazik ama mesafeli. Resmi dil kullanır. Güven arar.",
        patience_minutes=20,
    ),
    Persona(
        name="Ali Çelik",
        phone="905550001004",
        personality="Doğrudan alıcı. 'Bunu istiyorum, nasıl alırım?' der. Az konuşur.",
        patience_minutes=15,
    ),
    Persona(
        name="Zeynep Aksoy",
        phone="905550001005",
        personality="Emoji ve kısaltma kullanır. Genç, WhatsApp'a alışkın. hızlı yazar.",
        patience_minutes=12,
    ),
    Persona(
        name="Hasan Öztürk",
        phone="905550001006",
        personality="Şüpheci, 'bu gerçek mi?' diye sorar. İkna edilmesi gerekir.",
        patience_minutes=25,
    ),
    Persona(
        name="Elif Şahin",
        phone="905550001007",
        personality="Fiyat karşılaştırması yapar. 'Başka yerde daha ucuz' der.",
        patience_minutes=20,
    ),
    Persona(
        name="Burak Aydın",
        phone="905550001008",
        personality="Teknik sorular sorar. Ürün detaylarını, garanti şartlarını merak eder.",
        patience_minutes=30,
    ),
    Persona(
        name="Selin Koç",
        phone="905550001009",
        personality="Arkadaşça ve samimi. Sohbet eder, güler yüzlü. Ama karar vermesi zaman alır.",
        patience_minutes=35,
    ),
    Persona(
        name="Emre Yıldız",
        phone="905550001010",
        personality="Meşgul iş insanı. Sadece akşam yazışır. 'Kısa kesin' ister.",
        patience_minutes=8,
    ),
]
