# PhotoCurator — Metadaten-Strategie

## Zweck

Dieses Dokument hält fest, welche Metadatenfelder UGREEN Fotos je Format liest und wie
PhotoCurator Tags, Orte und Beschreibungen daher schreiben muss. Es entsteht aus dem Spike
(Story 001, Metadatenfelder von UGREEN Fotos) und konkretisiert die Metadaten-Strategie in
Tech Spec §3 (`photocurator-tech-spec.md`). Die Feldmatrix wird beim Spike durch die
Beobachtungen aus `PROTOKOLL.md` gefüllt; bis dahin gilt die vorläufige Strategie.

## Feldmatrix

Eine Zeile je Format und Feldschlüssel der Probe-Dateien (inklusive Sidecar-Varianten). Die
letzten vier Spalten sind im Spike (2026-10-30, UGREEN Fotos Desktop-App, ExifTool 12.86 auf dem NAS) ausgefüllt.

`*` = nicht einzeln geprüft. Geprüft wurden die zwölf JPEG-Textmarker (Suche) und `pc001` (Tag/Kategorie, Detailansicht);
die übrigen Formate sind aus dem Verhalten der JPEG-Dateien übertragen. `nicht geprüft` = das Orts-Feld im bedingten Album wurde nicht ausprobiert.

| Format | Feldkey | ExifTool-Tags | Suche | Tag/Kategorie | bedingtes Album | Ort angezeigt |
| --- | --- | --- | --- | --- | --- | --- |
| jpeg | xmpsubject | `XMP-dc:Subject` | nein | nein | nein | – |
| jpeg | hierarchical | `XMP-lr:HierarchicalSubject` | nein | nein | nein | – |
| jpeg | xmpdescription | `XMP-dc:Description` | nein | nein | nein | – |
| jpeg | exifdescription | `EXIF:ImageDescription` | nein | nein | nein | – |
| jpeg | gps-exif | `EXIF:GPSLatitude`, `EXIF:GPSLatitudeRef`, `EXIF:GPSLongitude`, `EXIF:GPSLongitudeRef` | ja (Ort) | ja (Locations) | nicht geprüft | ja |
| jpeg | gps-xmp | `XMP-exif:GPSLatitude`, `XMP-exif:GPSLongitude` | nein | nein | nicht geprüft | nein |
| jpeg | loc-photoshop | `XMP-photoshop:City`, `XMP-photoshop:State`, `XMP-photoshop:Country` | nein | nein | nein | nein |
| jpeg | loc-iptccore | `XMP-iptcCore:Location` | nein | nein | nein | nein |
| jpeg | loc-iptcext | `XMP-iptcExt:LocationShownCity`, `XMP-iptcExt:LocationShownProvinceState`, `XMP-iptcExt:LocationShownCountryName`, `XMP-iptcExt:LocationShownLocationName` | nein | nein | nein | nein |
| jpeg | iptckeywords | `IPTC:Keywords` | nein | nein | nein | – |
| jpeg | xpkeywords | `EXIF:XPKeywords` | nein | nein | nein | – |
| jpeg | iptccaption | `IPTC:Caption-Abstract` | nein | nein | nein | – |
| jpeg | loc-iptc | `IPTC:City`, `IPTC:Sub-location`, `IPTC:Province-State`, `IPTC:Country-PrimaryLocationName` | nein | nein | nein | nein |
| jpeg | combo | `XMP-dc:Subject`, `XMP-lr:HierarchicalSubject`, `XMP-dc:Description`, `EXIF:ImageDescription`, `EXIF:GPSLatitude`, `EXIF:GPSLatitudeRef`, `EXIF:GPSLongitude`, `EXIF:GPSLongitudeRef`, `XMP-exif:GPSLatitude`, `XMP-exif:GPSLongitude`, `XMP-photoshop:City`, `XMP-photoshop:State`, `XMP-photoshop:Country`, `XMP-iptcCore:Location`, `XMP-iptcExt:LocationShownCity`, `XMP-iptcExt:LocationShownProvinceState`, `XMP-iptcExt:LocationShownCountryName`, `XMP-iptcExt:LocationShownLocationName`, `IPTC:Keywords`, `EXIF:XPKeywords`, `IPTC:Caption-Abstract`, `IPTC:City`, `IPTC:Sub-location`, `IPTC:Province-State`, `IPTC:Country-PrimaryLocationName` | ja (nur Ort) | ja (Locations) | nicht geprüft | ja |
| heic | xmpsubject | `XMP-dc:Subject` | nein* | nein* | nein* | – |
| heic | hierarchical | `XMP-lr:HierarchicalSubject` | nein* | nein* | nein* | – |
| heic | xmpdescription | `XMP-dc:Description` | nein* | nein* | nein* | – |
| heic | exifdescription | `EXIF:ImageDescription` | nein* | nein* | nein* | – |
| heic | gps-exif | `EXIF:GPSLatitude`, `EXIF:GPSLatitudeRef`, `EXIF:GPSLongitude`, `EXIF:GPSLongitudeRef` | ja (Ort) | ja (Locations) | nicht geprüft | ja |
| heic | gps-xmp | `XMP-exif:GPSLatitude`, `XMP-exif:GPSLongitude` | nein | nein | nicht geprüft | nein |
| heic | loc-photoshop | `XMP-photoshop:City`, `XMP-photoshop:State`, `XMP-photoshop:Country` | nein* | nein* | nein* | nein* |
| heic | loc-iptccore | `XMP-iptcCore:Location` | nein* | nein* | nein* | nein* |
| heic | loc-iptcext | `XMP-iptcExt:LocationShownCity`, `XMP-iptcExt:LocationShownProvinceState`, `XMP-iptcExt:LocationShownCountryName`, `XMP-iptcExt:LocationShownLocationName` | nein* | nein* | nein* | nein* |
| heic | combo | `XMP-dc:Subject`, `XMP-lr:HierarchicalSubject`, `XMP-dc:Description`, `EXIF:ImageDescription`, `EXIF:GPSLatitude`, `EXIF:GPSLatitudeRef`, `EXIF:GPSLongitude`, `EXIF:GPSLongitudeRef`, `XMP-exif:GPSLatitude`, `XMP-exif:GPSLongitude`, `XMP-photoshop:City`, `XMP-photoshop:State`, `XMP-photoshop:Country`, `XMP-iptcCore:Location`, `XMP-iptcExt:LocationShownCity`, `XMP-iptcExt:LocationShownProvinceState`, `XMP-iptcExt:LocationShownCountryName`, `XMP-iptcExt:LocationShownLocationName` | ja (nur Ort) | ja (Locations) | nicht geprüft | ja |
| png | xmpsubject | `XMP-dc:Subject` | nein* | nein* | nein* | – |
| png | hierarchical | `XMP-lr:HierarchicalSubject` | nein* | nein* | nein* | – |
| png | xmpdescription | `XMP-dc:Description` | nein* | nein* | nein* | – |
| png | exifdescription | `EXIF:ImageDescription` | nein* | nein* | nein* | – |
| png | gps-exif | `EXIF:GPSLatitude`, `EXIF:GPSLatitudeRef`, `EXIF:GPSLongitude`, `EXIF:GPSLongitudeRef` | ja (Ort) | ja (Locations) | nicht geprüft | ja |
| png | gps-xmp | `XMP-exif:GPSLatitude`, `XMP-exif:GPSLongitude` | nein | nein | nicht geprüft | nein |
| png | loc-photoshop | `XMP-photoshop:City`, `XMP-photoshop:State`, `XMP-photoshop:Country` | nein* | nein* | nein* | nein* |
| png | loc-iptccore | `XMP-iptcCore:Location` | nein* | nein* | nein* | nein* |
| png | loc-iptcext | `XMP-iptcExt:LocationShownCity`, `XMP-iptcExt:LocationShownProvinceState`, `XMP-iptcExt:LocationShownCountryName`, `XMP-iptcExt:LocationShownLocationName` | nein* | nein* | nein* | nein* |
| png | combo | `XMP-dc:Subject`, `XMP-lr:HierarchicalSubject`, `XMP-dc:Description`, `EXIF:ImageDescription`, `EXIF:GPSLatitude`, `EXIF:GPSLatitudeRef`, `EXIF:GPSLongitude`, `EXIF:GPSLongitudeRef`, `XMP-exif:GPSLatitude`, `XMP-exif:GPSLongitude`, `XMP-photoshop:City`, `XMP-photoshop:State`, `XMP-photoshop:Country`, `XMP-iptcCore:Location`, `XMP-iptcExt:LocationShownCity`, `XMP-iptcExt:LocationShownProvinceState`, `XMP-iptcExt:LocationShownCountryName`, `XMP-iptcExt:LocationShownLocationName` | ja (nur Ort) | ja (Locations) | nicht geprüft | ja |
| mp4 | xmpsubject | `XMP-dc:Subject` | nein* | nein* | nein* | – |
| mp4 | hierarchical | `XMP-lr:HierarchicalSubject` | nein* | nein* | nein* | – |
| mp4 | xmpdescription | `XMP-dc:Description` | nein* | nein* | nein* | – |
| mp4 | keys-keywords | `Keys:Keywords` | nein* | nein* | nein* | – |
| mp4 | keys-description | `Keys:Description` | nein* | nein* | nein* | – |
| mp4 | gps-keys | `Keys:GPSCoordinates` | ja (Ort) | ja (Locations) | nicht geprüft | ja |
| mp4 | gps-userdata | `UserData:GPSCoordinates` | ja (Ort) | ja (Locations) | nicht geprüft | ja |
| mp4 | gps-xmp | `XMP-exif:GPSLatitude`, `XMP-exif:GPSLongitude` | ja (Ort) | ja (Locations) | nicht geprüft | ja |
| mp4 | loc-photoshop | `XMP-photoshop:City`, `XMP-photoshop:State`, `XMP-photoshop:Country` | nein* | nein* | nein* | nein* |
| mp4 | loc-iptccore | `XMP-iptcCore:Location` | nein* | nein* | nein* | nein* |
| mp4 | loc-iptcext | `XMP-iptcExt:LocationShownCity`, `XMP-iptcExt:LocationShownProvinceState`, `XMP-iptcExt:LocationShownCountryName`, `XMP-iptcExt:LocationShownLocationName` | nein* | nein* | nein* | nein* |
| mp4 | combo | `XMP-dc:Subject`, `XMP-lr:HierarchicalSubject`, `XMP-dc:Description`, `Keys:Keywords`, `Keys:Description`, `Keys:GPSCoordinates`, `UserData:GPSCoordinates`, `XMP-exif:GPSLatitude`, `XMP-exif:GPSLongitude`, `XMP-photoshop:City`, `XMP-photoshop:State`, `XMP-photoshop:Country`, `XMP-iptcCore:Location`, `XMP-iptcExt:LocationShownCity`, `XMP-iptcExt:LocationShownProvinceState`, `XMP-iptcExt:LocationShownCountryName`, `XMP-iptcExt:LocationShownLocationName` | ja (nur Ort) | ja (Locations) | nicht geprüft | ja |
| mp4 | sidecar-name | `XMP-dc:Subject`, `XMP-lr:HierarchicalSubject`, `XMP-dc:Description`, `XMP-photoshop:City` | nein* | nein* | nein* | nein* |
| mp4 | sidecar-ext | `XMP-dc:Subject`, `XMP-lr:HierarchicalSubject`, `XMP-dc:Description`, `XMP-photoshop:City` | nein* | nein* | nein* | nein* |
| mov | xmpsubject | `XMP-dc:Subject` | nein* | nein* | nein* | – |
| mov | hierarchical | `XMP-lr:HierarchicalSubject` | nein* | nein* | nein* | – |
| mov | xmpdescription | `XMP-dc:Description` | nein* | nein* | nein* | – |
| mov | keys-keywords | `Keys:Keywords` | nein* | nein* | nein* | – |
| mov | keys-description | `Keys:Description` | nein* | nein* | nein* | – |
| mov | gps-keys | `Keys:GPSCoordinates` | ja (Ort) | ja (Locations) | nicht geprüft | ja |
| mov | gps-userdata | `UserData:GPSCoordinates` | ja (Ort) | ja (Locations) | nicht geprüft | ja |
| mov | gps-xmp | `XMP-exif:GPSLatitude`, `XMP-exif:GPSLongitude` | ja (Ort) | ja (Locations) | nicht geprüft | ja |
| mov | loc-photoshop | `XMP-photoshop:City`, `XMP-photoshop:State`, `XMP-photoshop:Country` | nein* | nein* | nein* | nein* |
| mov | loc-iptccore | `XMP-iptcCore:Location` | nein* | nein* | nein* | nein* |
| mov | loc-iptcext | `XMP-iptcExt:LocationShownCity`, `XMP-iptcExt:LocationShownProvinceState`, `XMP-iptcExt:LocationShownCountryName`, `XMP-iptcExt:LocationShownLocationName` | nein* | nein* | nein* | nein* |
| mov | combo | `XMP-dc:Subject`, `XMP-lr:HierarchicalSubject`, `XMP-dc:Description`, `Keys:Keywords`, `Keys:Description`, `Keys:GPSCoordinates`, `UserData:GPSCoordinates`, `XMP-exif:GPSLatitude`, `XMP-exif:GPSLongitude`, `XMP-photoshop:City`, `XMP-photoshop:State`, `XMP-photoshop:Country`, `XMP-iptcCore:Location`, `XMP-iptcExt:LocationShownCity`, `XMP-iptcExt:LocationShownProvinceState`, `XMP-iptcExt:LocationShownCountryName`, `XMP-iptcExt:LocationShownLocationName` | ja (nur Ort) | ja (Locations) | nicht geprüft | ja |
| mov | sidecar-name | `XMP-dc:Subject`, `XMP-lr:HierarchicalSubject`, `XMP-dc:Description`, `XMP-photoshop:City` | nein* | nein* | nein* | nein* |
| mov | sidecar-ext | `XMP-dc:Subject`, `XMP-lr:HierarchicalSubject`, `XMP-dc:Description`, `XMP-photoshop:City` | nein* | nein* | nein* | nein* |
| jpeg | nometa | keine (alle Metadaten entfernt) | nein | nein | nein | – |

## Erkennung von Änderungen

Ergebnis des Erkennungstests. Statt `probe mutate` (Marker in Tags, in der App nicht sichtbar) wurde der **Ort** geändert (Paris → Tokyo Tower), je Datei:

| Datei | mtime | automatisch erkannt | erst nach manuellem Rescan | Verzögerung |
| --- | --- | --- | --- | --- |
| pcgps-keep.jpg | unverändert | ja (Ort wechselte von Paris auf Tokyo Tower) | ja, Aktualisieren war gedrückt; automatische Erkennung nicht gemessen | nicht gemessen |
| pcgps-new.jpg | neu | **nein** — Datei verschwand aus Index und App-Ordner, auch nach Aktualisieren | — | blieb weg, bis dieselbe Datei mit altem Datum erneut kopiert wurde (dann Ort Tokyo Tower) |

## Vorläufige Strategie (Stand vor dem Spike; Ergebnis siehe „Entscheidung nach Spike“)

Bis zum Spike-Ergebnis gilt Tech Spec §3.2 („Im Zweifel mehrfach schreiben"):

- **Tags, Bilder:** `XMP-dc:Subject` mit allen Blättern und Elternbegriffen;
  `XMP-lr:HierarchicalSubject` mit vollständigen Pfaden (`Anlass|Geburtstag`);
  `IPTC:Keywords` wie `dc:Subject` (nur JPEG, mit `IPTC:CodedCharacterSet=UTF8`, höchstens
  64 Byte je Eintrag); `EXIF:XPKeywords` wie `dc:Subject`, durch `;` getrennt (nur JPEG).
- **Tags, Videos:** `XMP-dc:Subject` und `XMP-lr:HierarchicalSubject` eingebettet;
  QuickTime-Keywords nur, falls UGREEN sie liest.
- **Ort:** GPS-Koordinaten (Bilder EXIF; Videos `Keys:GPSCoordinates` und
  `UserData:GPSCoordinates`) plus Textfelder in XMP-photoshop/iptcCore und (JPEG) IPTC;
  `loc_label` zusätzlich als `XMP-iptcExt:LocationShownLocationName`.
- **Prüfstatus:** nur `XMP-photocurator:*`, nie in Keyword-Feldern.
- **Fremde Werte bleiben erhalten:** gezieltes Hinzufügen (`+=`) und Entfernen (`-=`), nie
  Überschreiben ganzer Listen.
- **Lesen:** Tags aus allen Keyword-Feldern zusammenführen; Hierarchie hat Vorrang.
- Das Mapping liegt als eine Tabelle in `internal/metadata`, damit das Spike-Ergebnis an
  genau einer Stelle einfließt.

## Entscheidung nach Spike

Beobachtung (2026-10-30): UGREEN Fotos liest **nur GPS-Koordinaten** aus den Dateien und zeigt dazu einen selbst
per Reverse-Geocoding ermittelten Ortsnamen (Locations, Suche, Karte). Tags, Beschreibungen, Hierarchie und
Ortstextfelder (Photoshop, IPTC, XMP-iptcExt) werden weder in Suche noch Kategorien noch bedingten Alben genutzt;
das Stichwort-Feld bedingter Alben filtert nur Dateinamen. Die Files-App findet Metadaten-Marker ebenfalls nicht.
Die UGOS-Tags der Files-App (Grün, Rot, …) und Favoriten liegen **nicht in der Datei** (Datei blieb byte-identisch).

- **Felder je Format:**
  - Ort (wirkt in der App): Bilder (JPEG, HEIC, PNG) `EXIF:GPSLatitude/-Ref/GPSLongitude/-Ref`; Videos (MP4, MOV)
    `Keys:GPSCoordinates` und `UserData:GPSCoordinates` (beide gelesen; XMP-GPS in Videos wird ebenfalls gelesen).
    `XMP-exif`-GPS in Bildern wird **nicht** gelesen.
  - Ortstexte (`loc_label`, Photoshop/IPTC): ohne Wirkung in der App; sie dürfen als zusätzliche Information
    geschrieben werden, die App-Anzeige hängt aber allein an den Koordinaten.
  - Tags und Beschreibung: ohne Wirkung in UGREEN Fotos und Files-App. Sie werden gemäß „Vorläufige Strategie“
    trotzdem geschrieben (Wiederauffindbarkeit, Interoperabilität, PhotoCurator-eigener Index); eine Reduktion auf
    `XMP-dc:Subject` + `XMP-lr:HierarchicalSubject` ist möglich und kein Spike-Thema mehr.
- **Video-Sidecars (`.xmp`) schreiben:** **nein.** Sidecars werden von der App ignoriert (Machu Picchu fehlt in
  Locations); Videos tragen die Felder eingebettet.
- **Default für `PC_PRESERVE_MTIME`:** **`true`.** Eine Datei mit neuem Änderungsdatum verschwand nach dem
  Überschreiben aus Index und App-Ordner (auch nach Aktualisieren); mit unverändertem Datum wurde der neue Ort
  übernommen. Dieselbe Datei mit auf 2001-01-01 zurückgesetztem Datum erschien wieder. Offen: Ob die Erkennung
  ohne manuelles Aktualisieren von selbst geschieht und wie lange sie dauert, wurde nicht gemessen.
- **Regel, falls UGREEN selbst Metadaten schreibt:** UGREEN Fotos und Files-App schreiben nichts in Dateien
  (Favorit gesetzt, Files-Tag „Grün“ gesetzt: Datei jeweils byte-identisch). Fremde Änderungen sind daher nur
  von anderen Werkzeugen zu erwarten; die Regel „gezielt hinzufügen/entfernen, nie ganze Listen überschreiben“
  bleibt.
- **Bedingte Albumfelder:** People & Pets, Aufnahmezeit, Typ, Ort (Text), Objekterkennung, Gerät, Quelle,
  Stichwörter (nur Dateiname). Kein Feld liest Tags aus der Datei.
- **Folge für die Produktidee:** Tags in Dateien sind in UGREEN Fotos und Files-App nicht sichtbar; nur Orte
  (GPS) erscheinen dort. Der Nutzen von Tags liegt in PhotoCurator selbst.
