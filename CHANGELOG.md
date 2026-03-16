# Changelog

Bu dosya proje degisikliklerini versiyon bazli takip eder.
Format: [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), SemVer.

## [Unreleased]

### Added
- Ulke seviyesinde ekonomi simulasyonu icin yeni `go-sim/internal/econ` cekirdegi eklendi.
  4 ulke/4 sehir uzerinden `energy`, `compute`, `infrastructure`, kaynak cikarma, quote tabanli trade ve algorithm varyantlari (`no_trade`, `linear`, `scarcity_spike`) calisabiliyor.
- Yeni ekonomi server'i eklendi: `go-sim/cmd/omega-econ-server`.
  `/api/econ/state`, `/api/econ/step`, `/api/econ/reset`, `/api/econ/config`, `/api/econ/runs`, `/api/econ/compare`, `/api/econ/lab`, `/api/econ/experiments` endpoint'leri ile state, run, compare, lab session ve experiment sonuc servisleri saglaniyor.
- CLI-first experiment sistemi eklendi: `go-sim/cmd/omega-econ`.
  `experiment list`, `print`, `run`, `show` komutlari ile hipotez, control/variant senaryolar, bagimsiz degiskenler, ranking ve control comparison kayitlari JSON olarak uretilip `go-sim/experiment-results/` altina yaziliyor.
- Lab session archive yapisi eklendi.
  Compare/Lab session snapshot'lari `go-sim/lab-results/` altina kaydedilip sonradan tekrar incelenebiliyor.
- `POST /api/sim/replay` endpoint'i eklendi.
  Seed + config verilerek simulasyon deterministik sekilde ayni kosullarda tekrar kurulabiliyor.
  Mevcut run, replay baslangicindan once `replay_request` reason'i ile finalize ediliyor.
- Genetik policy tarafina hayatta kalma bootstrap nöronlari eklendi.
  Aksiyon secimi runtime kurali degil, tamamen genom skorlamasi + mutasyon temelli kaliyor.
  Baslangic populasyonu meyve toplama/yeme ve yakin meyveye gitme davranislarini genlerden ogrenebilir hale getirildi.
- Meyve, vitamin ve ureme temelli yeni klasik sim cekirdegi eklendi.
  `collect_fruit`, `eat_fruit`, `trade_fruit`, `clone_self` aksiyonlari; agac/meyve dagilimi, vitamin eksikligi, envanter dagilimi ve genom mutasyonu ile birlikte calisiyor.
- Klasik sim UI tarafina run history ve replay gorunumu eklendi.
  `History` sekmesi kaydedilmis run listesini, trace grafiklerini ve `POST /api/sim/replay` uzerinden ayni seed + config ile tekrar baslatmayi destekliyor.

### Changed
- Frontend girisi ekonomi workbench'i okuyacak sekilde guncellendi.
  `src/EconApp.tsx` ve ilgili hook/type dosyalari ile CLI'dan uretilen experiment sonuclari, compare/lab session'lari ve ekonomi state'i UI tarafinda okunabilir hale geldi.
- Vite dev server proxy ve npm script'leri ekonomi API'si ile birlikte calisacak sekilde guncellendi.
- Gecici experiment ve lab result artefact'lari `.gitignore` altina alindi.
- Klasik sim session yonetimi `speed` gibi UI-only config degisikliklerinde aktif run'i finalize etmeyecek sekilde duzeltildi.
- Klasik sim sidebar'i config/charts tab'lari, meyve/vitamin sparkline'lari ve alt grafik paneli ile genisletildi.

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
