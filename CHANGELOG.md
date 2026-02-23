# Changelog

Bu dosya proje degisikliklerini versiyon bazli takip eder.
Format: [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), SemVer.

## [Unreleased]

### Added
- `POST /api/sim/replay` endpoint'i eklendi.
  Seed + config verilerek simulasyon deterministik sekilde ayni kosullarda tekrar kurulabiliyor.
  Mevcut run, replay baslangicindan once `replay_request` reason'i ile finalize ediliyor.

### Planned
- Agent davranis/genetik degisikliklerinde run schema migration notlari.
- UI tarafinda run listesi ve run replay ekrani.

## [0.2.0] - 2026-02-23

### Added
- Go simulation server tarafinda kalici run kaydi (`go-sim/runs`).
- Run dosyalarina versiyon metadata alanlari:
  - `schemaVersion`
  - `serverVersion`
  - `apiVersion`
- Yeni endpoint: `GET /api/version`.
- Run endpoint cevabina versiyon bilgisi eklendi.

### Changed
- `GET /api/health` cevabina versiyon bilgisi eklendi.

## [0.1.0] - 2026-02-23

### Added
- Vite + React UI'nin Go backend ile entegre calismasi.
- `/api/sim/state`, `/api/sim/step`, `/api/sim/reset`, `/api/sim/config`, `/api/sim/agent/:id` akisi.
- GUI tarafinda backend baglanti durumu gorunurlugu.
