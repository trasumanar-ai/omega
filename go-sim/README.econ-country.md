# Omega Country Economy MVP

Bu paket, mevcut grid-world simden ayri duran yeni bir ekonomi cekirdegi ekler.

Model:

- 4 ulke / sehir
- sabit dogal kaynak dagilimi
- pasif extraction sistemleri
- ajan karariyla `energy`, `compute`, `infrastructure` yatirimi
- route uzerinden enerji ve build malzemesi ticareti
- compute, sadece enerji verimliligi olarak calisir

## Dunya Mantigi

Her ulke icin:

- kaynak yataklari: `coal`, `oil`, `copper`, `silicon`
- dogal potansiyeller: `sun`, `wind`
- pasif extractor: her tick stok uretir ve yatagi azaltir
- enerji varliklari: `coal_plant`, `oil_plant`, `solar_farm`, `wind_farm`
- altyapi: grid + trade kapasitesi
- compute: enerji ihtiyacini ve kayiplari azaltan verimlilik katsayisi

Tick akisi:

1. Ajan bir build focus secer: `hold`, `energy`, `compute`, `infrastructure`
2. Pasif extraction calisir
3. Yerel enerji uretimi yapilir
4. Route uzerinden enerji ve malzeme trade'i olur
5. Reserve, emergency buy ve shortage hesaplanir
6. Treasury/stability guncellenir
7. Uygun ise build uygulanir

## CLI

```bash
cd go-sim
go run ./cmd/omega-econ -steps 120 -sample-every 20
```

Opsiyonlar:

- `-steps`
- `-sample-every`
- `-out`

Rapor su alanlari doner:

- `final`: son tick aggregate metrikleri
- `world`: ulke ve route snapshot'i
- `trace`: zaman serisi

## HTTP Server

```bash
cd go-sim
go run ./cmd/omega-econ-server -port 8090
```

Endpoint'ler:

- `GET /api/health`
- `GET /api/econ/state`
- `POST /api/econ/step`
- `POST /api/econ/reset`
- `POST /api/econ/config`

`/api/econ/state` cevabi, daha sonra network-map UI cizmek icin yeterli veriyi doner:

- ulke koordinatlari (`x`, `y`)
- deposits / stockpiles / assets
- son tick enerji ve shortage bilgisi
- route kapasitesi ve son flow degerleri
- son karar kaynagi / model / token kullanimi

## LLM Provider

Server ve CLI, OpenRouter bagliysa heuristic yerine LLM kararlarini kullanir.

Gerekli env:

- `OPENROUTER_API_KEY`
- `ECON_LLM_MODEL` (varsayilan: `minimax/minimax-m2.5`)
- `ECON_LLM_MAX_TOKENS`
- `ECON_LLM_TEMPERATURE`
- `ECON_LLM_TIMEOUT_SECONDS`

LLM, ulke icin sadece bir `buildFocus` uretir:

- `energy`
- `compute`
- `infrastructure`
- `hold`

Karar metadata'si state icinde tutulur:

- `lastDecisionSource`
- `lastDecisionModel`
- `lastDecisionSummary`
- `lastDecisionError`
- `lastDecisionUsage`

## Notlar

- Bu ilk surumde savas yok.
- Ajan karar motoru su an heuristic; LLM action provider daha sonra bu arayuze takilabilir.
- Amac, dunyanin hemen cokmemesi ama kaynak azalmasi nedeniyle ajanlari trade ve transition'a zorlamasidir.
