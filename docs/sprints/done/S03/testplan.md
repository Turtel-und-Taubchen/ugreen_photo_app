# Testplan S03 — Manuelles Residuum

Nur Kriterien, die nicht automatisierbar sind. Kein Abnahme-Gate — die Tests sind es.

## 009 AC9 — UGREEN Fotos zeigt den geschriebenen Ort

**Grund:** UGREEN Fotos ist eine proprietäre App auf dem NAS und lässt sich nicht automatisieren.

**Vorbereitung:** PhotoCurator auf dem NAS laufen lassen (siehe `deploy/README.md`), `PC_PRESERVE_MTIME` auf dem Default `true` lassen. Ein Foto im Ordner `Sortiert/JJJJ/MM` ohne Ort wählen.

**Schritte:**
1. In PhotoCurator das Foto auswählen, über „Ort setzen" einen Ort (z. B. per Suche) setzen, Job abwarten.
2. In UGREEN Fotos „Aktualisieren" drücken.
3. Foto in UGREEN Fotos suchen (Locations/Karte) und die Detailansicht öffnen.

**Erwartet:** Das Foto bleibt sichtbar und zeigt den gesetzten Ort (Ortsname von UGREEN per Reverse-Geocoding, passend zu den Koordinaten).
