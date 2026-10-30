# Testdaten (Fixtures)

Kleine, selbst erzeugte Dateien ohne private Inhalte. Tests arbeiten nie auf diesen
Originalen, sondern immer auf einer Kopie (`testsupport.CopyFixture`).

| Datei | Inhalt | Herkunft |
| --- | --- | --- |
| `library/2024/07/sample.jpg` | 320×240, blaue Farbfläche mit Text „PhotoCurator Fixture“, JPEG Q80, ohne Metadaten | erzeugt mit libvips 8.16.1 (Debian trixie) |
| `library/2024/07/sample.heic` | dasselbe Bild als HEIC (x265, Q50), ohne Metadaten | erzeugt mit libvips 8.16.1 + `libheif-plugin-x265` 1.23.6 |
| `library/2024/07/tagged.jpg` | `sample.jpg` plus Metadaten: DateTimeOriginal 2024:07:14 10:00:00, dc:Subject Anlass/Geburtstag/Familie, lr:HierarchicalSubject `Anlass\|Geburtstag`, IPTC:Keywords Familie/Sommer (UTF-8), XPKeywords `Sommer;Garten`, GPS 47.0707 N 15.4395 E, photoshop City/State/Country Graz/Steiermark/Österreich, iptcCore:Location Stadtpark, iptcExt:LocationShownLocationName `Stadtpark, Graz`, photocurator:ReviewState `done` | ExifTool 13.25 aus `sample.jpg` |
| `library/2024/07/tagged.heic` | `sample.heic` plus Metadaten: DateTimeOriginal 2024:07:20 18:30:00, dc:Subject Reise, lr:HierarchicalSubject `Reise\|Österreich`, GPS 47.2692 N 11.4041 E, photoshop City Innsbruck, photocurator:ReviewState `irrelevant` | ExifTool 13.25 aus `sample.heic` |
| `library/2024/08/screenshot.png` | `sample.jpg` als PNG; nur dc:Subject Notizen, kein Datum (Quelle: mtime) | libvips + ExifTool |
| `library/2024/08/IMG-20240812-WA0001.jpg` | Kopie von `sample.jpg`, ohne Metadaten (Datum nur aus dem Namen) | Kopie |
| `library/2024/08/20240815_143005.jpg` | Kopie von `sample.jpg`, ohne Metadaten (Datum und Zeit nur aus dem Namen) | Kopie |
| `library/2024/08/clip.mp4` | 1 s, 64×64, H.264 ohne Ton, `moov` hinter `mdat` (kein faststart); QuickTime:CreateDate 2024:08:20 08:00:00 (UTC), dc:Subject Video, Keys:GPSCoordinates 48.2082 16.3738 | ffmpeg 7.1 + ExifTool, danach `moov` ans Ende verschoben |
| `library/2024/08/defekt.jpg` | 1 KB Zufallsbytes; `exiftool -j` liefert `Error` („File format error“) | `head -c 1024 /dev/urandom` |
| `library/2024/08/notizen.txt` | kurzer Text, kein Medium (muss ignoriert werden) | von Hand |

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

## Erzeugung der Metadaten-Fixtures

Die Dateien mit Metadaten entstehen aus `sample.*`. ExifTool bekommt dieselbe Konfiguration
wie die Anwendung (`-config internal/metadata/photocurator.config`), sonst ist der Namespace
`photocurator` unbekannt. ExifTool legt beim Schreiben das `moov`-Atom vor `mdat`; für `clip.mp4`
wird es danach mit dem Perl-Skript unten wieder ans Ende verschoben (stco-Offsets werden um die
Größe von `moov` verringert). Das Skript liegt als `/tmp/reorder.pl` vor und wird mit
`-v /tmp:/rp:ro` eingebunden.

```perl
# reorder.pl <datei.mp4>
undef $/; open F, "<:raw", $ARGV[0]; $d = <F>; close F; $p = 0;
while ($p < length $d) { $s = unpack("N", substr($d, $p, 4)); push @b, [substr($d, $p+4, 4), substr($d, $p, $s)]; $p += $s }
($mv) = grep { $_->[0] eq "moov" } @b; $z = length $mv->[1]; $m = $mv->[1]; $q = $m;
while ($q =~ /stco/g) { $o = pos($q) + 4; $c = unpack("N", substr($m, $o, 4)); $o += 4;
  for (1..$c) { substr($m, $o, 4) = pack("N", unpack("N", substr($m, $o, 4)) - $z); $o += 4 } }
open W, ">:raw", $ARGV[0]; print W $_->[1] for ((grep { $_->[0] ne "moov" } @b), [moov => $m]); close W;
```

```sh
docker run --rm -e DEBIAN_FRONTEND=noninteractive \
  -v "$(pwd)/testdata/library:/fx" -v "$(pwd)/internal/metadata:/cfg:ro" -v /tmp:/rp:ro \
  debian:trixie-slim sh -euc '
apt-get update -qq && apt-get install -y -qq --no-install-recommends libimage-exiftool-perl libvips-tools libheif-plugin-libde265 libheif-plugin-x265 ffmpeg >/dev/null
L=/fx/2024
C="exiftool -config /cfg/photocurator.config -overwrite_original"
cp $L/07/sample.jpg $L/07/tagged.jpg
$C -DateTimeOriginal="2024:07:14 10:00:00" -XMP-dc:Subject=Anlass -XMP-dc:Subject=Geburtstag -XMP-dc:Subject=Familie \
  -XMP-lr:HierarchicalSubject="Anlass|Geburtstag" -IPTC:CodedCharacterSet=UTF8 -IPTC:Keywords=Familie -IPTC:Keywords=Sommer \
  -EXIF:XPKeywords="Sommer;Garten" -GPSLatitude=47.0707 -GPSLatitudeRef=N -GPSLongitude=15.4395 -GPSLongitudeRef=E \
  -XMP-photoshop:City=Graz -XMP-photoshop:State=Steiermark -XMP-photoshop:Country="Österreich" \
  -XMP-iptcCore:Location=Stadtpark -XMP-iptcExt:LocationShownLocationName="Stadtpark, Graz" \
  -XMP-photocurator:ReviewState=done $L/07/tagged.jpg
cp $L/07/sample.heic $L/07/tagged.heic
$C -DateTimeOriginal="2024:07:20 18:30:00" -XMP-dc:Subject=Reise -XMP-lr:HierarchicalSubject="Reise|Österreich" \
  -GPSLatitude=47.2692 -GPSLatitudeRef=N -GPSLongitude=11.4041 -GPSLongitudeRef=E \
  -XMP-photoshop:City=Innsbruck -XMP-photocurator:ReviewState=irrelevant $L/07/tagged.heic
vips copy $L/07/sample.jpg $L/08/screenshot.png
$C -XMP-dc:Subject=Notizen $L/08/screenshot.png
cp $L/07/sample.jpg $L/08/IMG-20240812-WA0001.jpg
cp $L/07/sample.jpg $L/08/20240815_143005.jpg
ffmpeg -loglevel error -y -f lavfi -i color=c=blue:s=64x64:d=1:r=10 -c:v libx264 -pix_fmt yuv420p -an $L/08/clip.mp4
$C -QuickTime:CreateDate="2024:08:20 08:00:00" -XMP-dc:Subject=Video -Keys:GPSCoordinates="48.2082 16.3738" $L/08/clip.mp4
perl /rp/reorder.pl $L/08/clip.mp4
head -c 1024 /dev/urandom > $L/08/defekt.jpg
printf "Einkaufsliste: Brot, Milch\n" > $L/08/notizen.txt
exiftool -j $L/08/defekt.jpg | grep Error'
```
