# Frontend

UI: [CoreUI for Angular](https://github.com/coreui/coreui-angular) 5.6 + Bootstrap 5, **Angular 21**, **zoneless** (`provideZonelessChangeDetection`), состояние через **signals** (без reactive forms и без `zone.js` в production).

## Development server

```bash
npm install
npm start
```

Откройте `http://localhost:4200/`.

## Build

```bash
npm run build
```

Артефакты: `dist/frontend/browser/` (в Docker копируются в Go `go:embed`).

## Tests

```bash
npm test          # Vitest (watch)
npm run test:ci   # однократный прогон с coverage
```
