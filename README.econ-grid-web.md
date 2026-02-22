# econ-grid-web

Vite + TypeScript + Canvas API ile yazilmis grid tabanli simulasyon MVP'si.

## Ne var?
- Grid uzerinde tek hucre kaplayan ajanlar
- 4-yonlu komsulukta random hareket
- Tick bazli calisma (start/pause/step)
- Canvas uzerinde canli render + temel metrikler

## Calistirma
```bash
npm install
npm run dev
```

## Build
```bash
npm run build
npm run preview
```

## Sonraki adimlar
- Ekonomi katmani: uretim/tuketim, al-sat emirleri, fiyat olusumu
- Agent stratejileri: policy tabanli karar mekanizmasi
- Persistence: tick metriklerini disari alma (JSON/CSV)
- Backend ayirma: gerekirse Go servis katmanina gecis
