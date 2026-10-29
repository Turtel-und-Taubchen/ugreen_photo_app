package probe

import (
	"os"
	"path/filepath"
	"strings"
)

// ProtocolFile is the observation sheet written next to the probe files.
const ProtocolFile = "PROTOKOLL.md"

const protocolHeader = `# PROTOKOLL - UGREEN-Fotos-Metadaten-Probe

## Zweck

Die Probe-Dateien in diesem Ordner tragen je ein eindeutiges Marker-Token in genau einem
Metadatenfeld. Damit wird beobachtet, welche Felder UGREEN Fotos indiziert, in der Suche
findet, als Tag/Kategorie zeigt, in bedingten Alben nutzt und als Ort anzeigt.

## Ablauf

1. Ordner auf das NAS nach ` + "`Sortiert/_pcprobe/`" + ` kopieren.
2. Indizierung abwarten (Zeit bis zur Sichtbarkeit notieren).
3. Jede Zeile der Tabelle prüfen und die fünf leeren Spalten ausfüllen.
4. Den Ordner danach wieder löschen.

Erzeugung (Container):

` + "```\ndocker run --rm -v \"<ordner>:/out\" photocurator:dev probe generate --out /out\n```" + `

## Orte (GPS-Varianten)

| Wahrzeichen | GPS-Variante |
| --- | --- |
| Eiffelturm | EXIF-GPS |
| Kolosseum | XMP-exif-GPS |
| Sydney Opera House | Keys:GPSCoordinates |
| Freiheitsstatue | UserData:GPSCoordinates |
| Taj Mahal | Kombi-Datei (alle Felder) |
| Machu Picchu | Sidecar |

## Hinweise zur Suche

- OCR-Kontrolle: Findet die Suche ` + "`pcprobe-jpeg-nometa`" + `, liest UGREEN Text in Bildern. Suchtreffer
  nur zählen, wenn diese Kontrolle negativ ist.
- Hierarchie-Marker sowohl mit der Suche nach ` + "`pcprobe`" + ` als auch nach ` + "`hierarchical`" + ` prüfen.

## Tabelle

| Datei | Format | Feld | Marker (bzw. Ort) | in Suche gefunden | als Tag/Kategorie sichtbar | in bedingtem Album nutzbar | Ort angezeigt | Zeit bis sichtbar |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
`

const protocolFooter = `
## Erkennungstest

Nach der Indizierung im Ordner ausführen (Container analog zu oben):

` + "```\ndocker run --rm -v \"<ordner>:/out\" photocurator:dev probe mutate --dir /out\n```" + `

Das setzt in beiden Mutate-Dateien einen neuen Marker (v2). Pro Datei notieren, ob der neue
Marker automatisch gefunden wird, erst nach manuellem Rescan oder nach welcher Verzögerung:

- ` + "`pcmut-keep.jpg`" + `: Änderungsdatum (mtime) bleibt erhalten.
- ` + "`pcmut-new.jpg`" + `: erhält ein neues Änderungsdatum.

| Datei | mtime | automatisch gefunden | erst nach manuellem Rescan | Verzögerung |
| --- | --- | --- | --- | --- |
| pcmut-keep.jpg | unverändert | | | |
| pcmut-new.jpg | neu | | | |

Ergebnis liefert den Default für ` + "`PC_PRESERVE_MTIME`" + `.

## Prüffragen

1. Welche Bedingungen bieten bedingte Alben in UGREEN Fotos überhaupt an? Antwort:
2. Schreibt UGREEN Fotos selbst Metadaten in Dateien (z. B. beim Taggen oder Favorisieren in
   der App)? Vorgehen: ` + "`exiftool -a -G1 <datei>`" + ` vorher sichern, eine Probe-Datei in der App taggen
   bzw. favorisieren, erneut ` + "`exiftool -a -G1 <datei>`" + ` ausführen und beide Ausgaben vergleichen.
   Antwort:
`

// protocolRow is one table row: a file (or file + sidecar pair) and what it carries.
type protocolRow struct{ File, Format, Field, Marker string }

func protocolRows() []protocolRow {
	var out []protocolRow
	for _, p := range Probes() {
		file := p.File
		if p.Sidecar != "" {
			file += " + " + p.Sidecar
		}
		marker := p.Marker
		if strings.HasPrefix(p.FieldKey, "gps-") {
			marker = p.Place.Name
		}
		out = append(out, protocolRow{file, string(p.Format), p.FieldKey, marker})
	}
	for _, m := range MutateFiles() {
		out = append(out, protocolRow{m.File, string(FormatJPEG), "mutate", m.Marker})
	}
	return out
}

// Protocol renders the German observation sheet for all generated files.
func Protocol() string {
	var b strings.Builder
	b.WriteString(protocolHeader)
	for _, r := range protocolRows() {
		b.WriteString("| " + strings.Join([]string{r.File, r.Format, r.Field, r.Marker}, " | ") + " | | | | | |\n")
	}
	b.WriteString(protocolFooter)
	return b.String()
}

func generateProtocol(outDir string) error {
	return os.WriteFile(filepath.Join(outDir, ProtocolFile), []byte(Protocol()), 0o644)
}
