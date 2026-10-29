# Testdaten (Fixtures)

Kleine, selbst erzeugte Dateien ohne private Inhalte. Tests arbeiten nie auf diesen
Originalen, sondern immer auf einer Kopie (`testsupport.CopyFixture`).

| Datei | Inhalt | Herkunft |
| --- | --- | --- |
| `library/2024/07/sample.jpg` | 320×240, blaue Farbfläche mit Text „PhotoCurator Fixture“, JPEG Q80, ohne Metadaten | erzeugt mit libvips 8.16.1 (Debian trixie) |
| `library/2024/07/sample.heic` | dasselbe Bild als HEIC (x265, Q50), ohne Metadaten | erzeugt mit libvips 8.16.1 + `libheif-plugin-x265` 1.23.6 |

Das Laufzeit-Image enthält nur das Decoder-Plugin `libheif-plugin-libde265`. Zum Erzeugen
der HEIC-Datei braucht es zusätzlich den Encoder `libheif-plugin-x265`, deshalb entstehen
beide Dateien einmalig in einem Wegwerf-Container.

## Erzeugung

Aus dem Repository-Wurzelverzeichnis (Git Bash unter Windows: `MSYS_NO_PATHCONV=1` voranstellen
und `$(pwd -W)` statt `$(pwd)` verwenden):

```sh
docker run --rm -e DEBIAN_FRONTEND=noninteractive \
  -v "$(pwd)/testdata/library/2024/07:/out" debian:trixie-slim sh -euc '
apt-get update -qq && apt-get install -y -qq --no-install-recommends libvips-tools libheif-plugin-x265 libheif-plugin-libde265 fonts-dejavu-core >/dev/null
cd /tmp
vips black black.v 320 240 --bands 3
vips linear black.v bg.v "1 1 1" "40 110 180" --uchar
vips linear black.v white.v "1 1 1" "255 255 255" --uchar
vips text text.v "PhotoCurator Fixture" --font "DejaVu Sans Bold 18" --dpi 72
vips gravity text.v mask.v centre 320 240
vips ifthenelse mask.v white.v bg.v blend.v --blend
vips copy blend.v out.v --interpretation srgb
vips copy out.v "/out/sample.jpg[Q=80,strip]"
vips copy out.v "/out/sample.heic[Q=50,strip]"
vipsheader /out/sample.jpg /out/sample.heic'
```

Neu erzeugte Dateien sind nicht bitgleich (Encoder-Versionen); Tests dürfen deshalb nicht
auf Hashes der Originaldateien prüfen.
