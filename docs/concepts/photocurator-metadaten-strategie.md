# PhotoCurator — Metadaten-Strategie

## Zweck

Dieses Dokument hält fest, welche Metadatenfelder UGREEN Fotos je Format liest und wie
PhotoCurator Tags, Orte und Beschreibungen daher schreiben muss. Es entsteht aus dem Spike
(Story 001, Metadatenfelder von UGREEN Fotos) und konkretisiert die Metadaten-Strategie in
Tech Spec §3 (`photocurator-tech-spec.md`). Die Feldmatrix wird beim Spike durch die
Beobachtungen aus `PROTOKOLL.md` gefüllt; bis dahin gilt die vorläufige Strategie.

## Feldmatrix

Eine Zeile je Format und Feldschlüssel der Probe-Dateien (inklusive Sidecar-Varianten). Die
letzten vier Spalten werden im Spike ausgefüllt.

| Format | Feldkey | ExifTool-Tags | Suche | Tag/Kategorie | bedingtes Album | Ort angezeigt |
| --- | --- | --- | --- | --- | --- | --- |
| jpeg | xmpsubject | `XMP-dc:Subject` | | | | |
| jpeg | hierarchical | `XMP-lr:HierarchicalSubject` | | | | |
| jpeg | xmpdescription | `XMP-dc:Description` | | | | |
| jpeg | exifdescription | `EXIF:ImageDescription` | | | | |
| jpeg | gps-exif | `EXIF:GPSLatitude`, `EXIF:GPSLatitudeRef`, `EXIF:GPSLongitude`, `EXIF:GPSLongitudeRef` | | | | |
| jpeg | gps-xmp | `XMP-exif:GPSLatitude`, `XMP-exif:GPSLongitude` | | | | |
| jpeg | loc-photoshop | `XMP-photoshop:City`, `XMP-photoshop:State`, `XMP-photoshop:Country` | | | | |
| jpeg | loc-iptccore | `XMP-iptcCore:Location` | | | | |
| jpeg | loc-iptcext | `XMP-iptcExt:LocationShownCity`, `XMP-iptcExt:LocationShownProvinceState`, `XMP-iptcExt:LocationShownCountryName`, `XMP-iptcExt:LocationShownLocationName` | | | | |
| jpeg | iptckeywords | `IPTC:Keywords` | | | | |
| jpeg | xpkeywords | `EXIF:XPKeywords` | | | | |
| jpeg | iptccaption | `IPTC:Caption-Abstract` | | | | |
| jpeg | loc-iptc | `IPTC:City`, `IPTC:Sub-location`, `IPTC:Province-State`, `IPTC:Country-PrimaryLocationName` | | | | |
| jpeg | combo | `XMP-dc:Subject`, `XMP-lr:HierarchicalSubject`, `XMP-dc:Description`, `EXIF:ImageDescription`, `EXIF:GPSLatitude`, `EXIF:GPSLatitudeRef`, `EXIF:GPSLongitude`, `EXIF:GPSLongitudeRef`, `XMP-exif:GPSLatitude`, `XMP-exif:GPSLongitude`, `XMP-photoshop:City`, `XMP-photoshop:State`, `XMP-photoshop:Country`, `XMP-iptcCore:Location`, `XMP-iptcExt:LocationShownCity`, `XMP-iptcExt:LocationShownProvinceState`, `XMP-iptcExt:LocationShownCountryName`, `XMP-iptcExt:LocationShownLocationName`, `IPTC:Keywords`, `EXIF:XPKeywords`, `IPTC:Caption-Abstract`, `IPTC:City`, `IPTC:Sub-location`, `IPTC:Province-State`, `IPTC:Country-PrimaryLocationName` | | | | |
| heic | xmpsubject | `XMP-dc:Subject` | | | | |
| heic | hierarchical | `XMP-lr:HierarchicalSubject` | | | | |
| heic | xmpdescription | `XMP-dc:Description` | | | | |
| heic | exifdescription | `EXIF:ImageDescription` | | | | |
| heic | gps-exif | `EXIF:GPSLatitude`, `EXIF:GPSLatitudeRef`, `EXIF:GPSLongitude`, `EXIF:GPSLongitudeRef` | | | | |
| heic | gps-xmp | `XMP-exif:GPSLatitude`, `XMP-exif:GPSLongitude` | | | | |
| heic | loc-photoshop | `XMP-photoshop:City`, `XMP-photoshop:State`, `XMP-photoshop:Country` | | | | |
| heic | loc-iptccore | `XMP-iptcCore:Location` | | | | |
| heic | loc-iptcext | `XMP-iptcExt:LocationShownCity`, `XMP-iptcExt:LocationShownProvinceState`, `XMP-iptcExt:LocationShownCountryName`, `XMP-iptcExt:LocationShownLocationName` | | | | |
| heic | combo | `XMP-dc:Subject`, `XMP-lr:HierarchicalSubject`, `XMP-dc:Description`, `EXIF:ImageDescription`, `EXIF:GPSLatitude`, `EXIF:GPSLatitudeRef`, `EXIF:GPSLongitude`, `EXIF:GPSLongitudeRef`, `XMP-exif:GPSLatitude`, `XMP-exif:GPSLongitude`, `XMP-photoshop:City`, `XMP-photoshop:State`, `XMP-photoshop:Country`, `XMP-iptcCore:Location`, `XMP-iptcExt:LocationShownCity`, `XMP-iptcExt:LocationShownProvinceState`, `XMP-iptcExt:LocationShownCountryName`, `XMP-iptcExt:LocationShownLocationName` | | | | |
| png | xmpsubject | `XMP-dc:Subject` | | | | |
| png | hierarchical | `XMP-lr:HierarchicalSubject` | | | | |
| png | xmpdescription | `XMP-dc:Description` | | | | |
| png | exifdescription | `EXIF:ImageDescription` | | | | |
| png | gps-exif | `EXIF:GPSLatitude`, `EXIF:GPSLatitudeRef`, `EXIF:GPSLongitude`, `EXIF:GPSLongitudeRef` | | | | |
| png | gps-xmp | `XMP-exif:GPSLatitude`, `XMP-exif:GPSLongitude` | | | | |
| png | loc-photoshop | `XMP-photoshop:City`, `XMP-photoshop:State`, `XMP-photoshop:Country` | | | | |
| png | loc-iptccore | `XMP-iptcCore:Location` | | | | |
| png | loc-iptcext | `XMP-iptcExt:LocationShownCity`, `XMP-iptcExt:LocationShownProvinceState`, `XMP-iptcExt:LocationShownCountryName`, `XMP-iptcExt:LocationShownLocationName` | | | | |
| png | combo | `XMP-dc:Subject`, `XMP-lr:HierarchicalSubject`, `XMP-dc:Description`, `EXIF:ImageDescription`, `EXIF:GPSLatitude`, `EXIF:GPSLatitudeRef`, `EXIF:GPSLongitude`, `EXIF:GPSLongitudeRef`, `XMP-exif:GPSLatitude`, `XMP-exif:GPSLongitude`, `XMP-photoshop:City`, `XMP-photoshop:State`, `XMP-photoshop:Country`, `XMP-iptcCore:Location`, `XMP-iptcExt:LocationShownCity`, `XMP-iptcExt:LocationShownProvinceState`, `XMP-iptcExt:LocationShownCountryName`, `XMP-iptcExt:LocationShownLocationName` | | | | |
| mp4 | xmpsubject | `XMP-dc:Subject` | | | | |
| mp4 | hierarchical | `XMP-lr:HierarchicalSubject` | | | | |
| mp4 | xmpdescription | `XMP-dc:Description` | | | | |
| mp4 | keys-keywords | `Keys:Keywords` | | | | |
| mp4 | keys-description | `Keys:Description` | | | | |
| mp4 | gps-keys | `Keys:GPSCoordinates` | | | | |
| mp4 | gps-userdata | `UserData:GPSCoordinates` | | | | |
| mp4 | gps-xmp | `XMP-exif:GPSLatitude`, `XMP-exif:GPSLongitude` | | | | |
| mp4 | loc-photoshop | `XMP-photoshop:City`, `XMP-photoshop:State`, `XMP-photoshop:Country` | | | | |
| mp4 | loc-iptccore | `XMP-iptcCore:Location` | | | | |
| mp4 | loc-iptcext | `XMP-iptcExt:LocationShownCity`, `XMP-iptcExt:LocationShownProvinceState`, `XMP-iptcExt:LocationShownCountryName`, `XMP-iptcExt:LocationShownLocationName` | | | | |
| mp4 | combo | `XMP-dc:Subject`, `XMP-lr:HierarchicalSubject`, `XMP-dc:Description`, `Keys:Keywords`, `Keys:Description`, `Keys:GPSCoordinates`, `UserData:GPSCoordinates`, `XMP-exif:GPSLatitude`, `XMP-exif:GPSLongitude`, `XMP-photoshop:City`, `XMP-photoshop:State`, `XMP-photoshop:Country`, `XMP-iptcCore:Location`, `XMP-iptcExt:LocationShownCity`, `XMP-iptcExt:LocationShownProvinceState`, `XMP-iptcExt:LocationShownCountryName`, `XMP-iptcExt:LocationShownLocationName` | | | | |
| mp4 | sidecar-name | `XMP-dc:Subject`, `XMP-lr:HierarchicalSubject`, `XMP-dc:Description`, `XMP-photoshop:City` | | | | |
| mp4 | sidecar-ext | `XMP-dc:Subject`, `XMP-lr:HierarchicalSubject`, `XMP-dc:Description`, `XMP-photoshop:City` | | | | |
| mov | xmpsubject | `XMP-dc:Subject` | | | | |
| mov | hierarchical | `XMP-lr:HierarchicalSubject` | | | | |
| mov | xmpdescription | `XMP-dc:Description` | | | | |
| mov | keys-keywords | `Keys:Keywords` | | | | |
| mov | keys-description | `Keys:Description` | | | | |
| mov | gps-keys | `Keys:GPSCoordinates` | | | | |
| mov | gps-userdata | `UserData:GPSCoordinates` | | | | |
| mov | gps-xmp | `XMP-exif:GPSLatitude`, `XMP-exif:GPSLongitude` | | | | |
| mov | loc-photoshop | `XMP-photoshop:City`, `XMP-photoshop:State`, `XMP-photoshop:Country` | | | | |
| mov | loc-iptccore | `XMP-iptcCore:Location` | | | | |
| mov | loc-iptcext | `XMP-iptcExt:LocationShownCity`, `XMP-iptcExt:LocationShownProvinceState`, `XMP-iptcExt:LocationShownCountryName`, `XMP-iptcExt:LocationShownLocationName` | | | | |
| mov | combo | `XMP-dc:Subject`, `XMP-lr:HierarchicalSubject`, `XMP-dc:Description`, `Keys:Keywords`, `Keys:Description`, `Keys:GPSCoordinates`, `UserData:GPSCoordinates`, `XMP-exif:GPSLatitude`, `XMP-exif:GPSLongitude`, `XMP-photoshop:City`, `XMP-photoshop:State`, `XMP-photoshop:Country`, `XMP-iptcCore:Location`, `XMP-iptcExt:LocationShownCity`, `XMP-iptcExt:LocationShownProvinceState`, `XMP-iptcExt:LocationShownCountryName`, `XMP-iptcExt:LocationShownLocationName` | | | | |
| mov | sidecar-name | `XMP-dc:Subject`, `XMP-lr:HierarchicalSubject`, `XMP-dc:Description`, `XMP-photoshop:City` | | | | |
| mov | sidecar-ext | `XMP-dc:Subject`, `XMP-lr:HierarchicalSubject`, `XMP-dc:Description`, `XMP-photoshop:City` | | | | |
| jpeg | nometa | keine (alle Metadaten entfernt) | | | | |

## Erkennung von Änderungen

Ergebnis des Erkennungstests (`probe mutate`), je Datei:

| Datei | mtime | automatisch erkannt | erst nach manuellem Rescan | Verzögerung |
| --- | --- | --- | --- | --- |
| pcmut-keep.jpg | unverändert | | | |
| pcmut-new.jpg | neu | | | |

## Vorläufige Strategie

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

- Felder je Format (Bild, HEIC, PNG, Video): _offen_
- Video-Sidecars (`.xmp`) schreiben: ja / nein — _offen_
- Default für `PC_PRESERVE_MTIME`: _offen_
- Regel, falls UGREEN selbst Metadaten schreibt: _offen_
