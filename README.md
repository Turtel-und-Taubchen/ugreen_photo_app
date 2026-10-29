# PhotoCurator

## Voraussetzungen

- Node.js (mit npm)
- Docker

Go, ExifTool, libvips und ffmpeg sind lokal nicht installiert. Go-Build und
Go-Tests laufen daher ausschliesslich in Docker (Build-Stages `test` und `lint`).

## Einmalig

```
npm --prefix web ci
```

## Verifizieren

```
npm --prefix web run build && docker build --target runtime -t photocurator:dev .
docker build --target test . && npm --prefix web run test
docker build --target lint . && npm --prefix web run lint
npm --prefix web run check
npm --prefix web run test:e2e
```

Reihenfolge: build, test, lint, typecheck, e2e.
