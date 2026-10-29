// Package probe generates and evaluates the metadata probe files: one file per format and
// candidate field, each carrying a unique marker only in its target field.
package probe

import (
	"fmt"
	"strconv"
	"strings"
)

// Format is the container format of a probe file.
type Format string

const (
	FormatJPEG Format = "jpeg"
	FormatHEIC Format = "heic"
	FormatPNG  Format = "png"
	FormatMP4  Format = "mp4"
	FormatMOV  Format = "mov"
)

// KeyNoMeta is the field key of the WhatsApp-like control file without any metadata.
const KeyNoMeta = "nometa"

// KeyCombo is the field key of the file that carries every field of its format at once.
const KeyCombo = "combo"

// TagValue is one ExifTool assignment (tag name as written, with group prefix).
type TagValue struct {
	Tag   string
	Value string
}

// Place is a named coordinate; the zero value means "no GPS".
type Place struct {
	Name     string
	Lat, Lon float64
}

// Probe describes one generated file. Generator, read-back, protocol and doc-consistency
// tests all read this table, so a new row (e.g. video) automatically reaches all of them.
type Probe struct {
	File string
	// Sidecar is the file name of the XMP sidecar next to File ("" if none). For sidecar
	// rows Texts and GPS live in the sidecar, never in File.
	Sidecar  string
	Format   Format
	FieldKey string
	// Tags lists every tag the file writes for its target field (text and coordinates).
	Tags []string
	// Marker identifies the probe in the pixels and in the protocol; the text values in
	// Texts derive from it (location subfields append a suffix, combo prefixes the key).
	Marker string
	Place  Place
	// Texts carry a marker each; GPS holds coordinate assignments (no text marker).
	Texts []TagValue
	GPS   []TagValue
	// Fixed are required companion assignments that carry no marker (e.g. IPTC charset).
	Fixed []TagValue
}

// PixelText is the text rendered into the picture: the marker, or for GPS files the field
// key and place name (their metadata holds no text marker).
func (p Probe) PixelText() string {
	if strings.HasPrefix(p.FieldKey, "gps-") {
		return p.FieldKey + "\n" + p.Place.Name
	}
	return p.Marker
}

type tagDef struct {
	tag          string
	sub          string // marker suffix for location subfields
	hierarchical bool
}

type gpsKind int

const (
	gpsNone gpsKind = iota
	gpsEXIF
	gpsXMP
	gpsKeys
	gpsUserData
)

type fieldDef struct {
	key      string
	jpegOnly bool
	tags     []tagDef
	gps      gpsKind
	place    Place
	fixed    []TagValue
}

var (
	eiffel = Place{Name: "Eiffel tower", Lat: 48.8584, Lon: 2.2945}
	colos  = Place{Name: "Colosseum", Lat: 41.8902, Lon: 12.4922}
	taj    = Place{Name: "Taj Mahal", Lat: 27.1751, Lon: 78.0421}
	sydney = Place{Name: "Sydney Opera House", Lat: -33.8568, Lon: 151.2153}
	nyc    = Place{Name: "Statue of Liberty", Lat: 40.6892, Lon: -74.0445}
	machu  = Place{Name: "Machu Picchu", Lat: -13.1631, Lon: -72.5450}

	iptcUTF8 = []TagValue{{"IPTC:CodedCharacterSet", "UTF8"}}
)

// fieldDefs lists the candidate metadata fields. IPTC text fields are capped at 32 bytes
// (city, sub-location, province), hence the short subfield suffixes of the loc-iptc row.
// ExifTool cannot write IPTC into PNG/HEIC reliably, so the IPTC and XP rows are JPEG-only.
var fieldDefs = []fieldDef{
	{key: "xmpsubject", tags: []tagDef{{tag: "XMP-dc:Subject"}}},
	{key: "hierarchical", tags: []tagDef{{tag: "XMP-lr:HierarchicalSubject", hierarchical: true}}},
	{key: "xmpdescription", tags: []tagDef{{tag: "XMP-dc:Description"}}},
	{key: "exifdescription", tags: []tagDef{{tag: "EXIF:ImageDescription"}}},
	{key: "gps-exif", gps: gpsEXIF, place: eiffel},
	{key: "gps-xmp", gps: gpsXMP, place: colos},
	{key: "loc-photoshop", tags: []tagDef{
		{tag: "XMP-photoshop:City", sub: "city"},
		{tag: "XMP-photoshop:State", sub: "state"},
		{tag: "XMP-photoshop:Country", sub: "country"}}},
	{key: "loc-iptccore", tags: []tagDef{{tag: "XMP-iptcCore:Location"}}},
	{key: "loc-iptcext", tags: []tagDef{
		{tag: "XMP-iptcExt:LocationShownCity", sub: "city"},
		{tag: "XMP-iptcExt:LocationShownProvinceState", sub: "provincestate"},
		{tag: "XMP-iptcExt:LocationShownCountryName", sub: "countryname"},
		{tag: "XMP-iptcExt:LocationShownLocationName", sub: "locationname"}}},
	{key: "iptckeywords", jpegOnly: true, fixed: iptcUTF8, tags: []tagDef{{tag: "IPTC:Keywords"}}},
	{key: "xpkeywords", jpegOnly: true, tags: []tagDef{{tag: "EXIF:XPKeywords"}}},
	{key: "iptccaption", jpegOnly: true, fixed: iptcUTF8, tags: []tagDef{{tag: "IPTC:Caption-Abstract"}}},
	{key: "loc-iptc", jpegOnly: true, fixed: iptcUTF8, tags: []tagDef{
		{tag: "IPTC:City", sub: "city"},
		{tag: "IPTC:Sub-location", sub: "sub"},
		{tag: "IPTC:Province-State", sub: "st"},
		{tag: "IPTC:Country-PrimaryLocationName", sub: "country"}}},
}

// Sidecar variants: the XMP sidecar is named after the video stem or after the full file name.
const (
	VariantSidecarName = "sidecar-name"
	VariantSidecarExt  = "sidecar-ext"
)

var videoFormats = []Format{FormatMP4, FormatMOV}

// sidecarDefs are the XMP fields a sidecar carries (City only for location).
var sidecarDefs = []fieldDef{
	{key: "xmpsubject", tags: []tagDef{{tag: "XMP-dc:Subject"}}},
	{key: "hierarchical", tags: []tagDef{{tag: "XMP-lr:HierarchicalSubject", hierarchical: true}}},
	{key: "xmpdescription", tags: []tagDef{{tag: "XMP-dc:Description"}}},
	{key: "loc-photoshop", tags: []tagDef{{tag: "XMP-photoshop:City"}}},
}

// videoDefs lists the embedded video fields. UserData:Keywords has no row: ExifTool 13.25
// reports it as not writable for both MP4 and MOV.
func videoDefs() []fieldDef {
	byKey := map[string]fieldDef{}
	for _, d := range fieldDefs {
		byKey[d.key] = d
	}
	return []fieldDef{
		byKey["xmpsubject"], byKey["hierarchical"], byKey["xmpdescription"],
		{key: "keys-keywords", tags: []tagDef{{tag: "Keys:Keywords"}}},
		{key: "keys-description", tags: []tagDef{{tag: "Keys:Description"}}},
		{key: "gps-keys", gps: gpsKeys, place: sydney},
		{key: "gps-userdata", gps: gpsUserData, place: nyc},
		byKey["gps-xmp"], byKey["loc-photoshop"], byKey["loc-iptccore"], byKey["loc-iptcext"],
	}
}

// imageFormats is the order in which image probes are numbered.
var imageFormats = []Format{FormatJPEG, FormatHEIC, FormatPNG}

// WhatsAppFile is the control file whose date exists only in its name and mtime.
const WhatsAppFile = "IMG-20010101-WA0001.jpg"

var probes = buildProbes()

// Probes returns the fixed probe table (a copy; callers may reorder it).
func Probes() []Probe {
	return append([]Probe(nil), probes...)
}

func buildProbes() []Probe {
	var out []Probe
	n := 0
	next := func(f Format) string {
		n++
		return fmt.Sprintf("pc%03d.%s", n, f)
	}
	for _, f := range imageFormats {
		var defs []fieldDef
		for _, d := range fieldDefs {
			if !d.jpegOnly || f == FormatJPEG {
				defs = append(defs, d)
			}
		}
		out = append(out, rows(f, defs, next)...)
	}
	for _, f := range videoFormats {
		out = append(out, rows(f, videoDefs(), next)...)
		for _, variant := range []string{VariantSidecarName, VariantSidecarExt} {
			out = append(out, sidecarRow(f, variant, next(f)))
		}
	}
	out = append(out, Probe{File: WhatsAppFile, Format: FormatJPEG, FieldKey: KeyNoMeta, Marker: "pcprobe-jpeg-" + KeyNoMeta})
	return out
}

// rows builds one probe per field definition plus the combo probe of one format.
func rows(f Format, defs []fieldDef, next func(Format) string) []Probe {
	var out []Probe
	for _, d := range defs {
		p := Probe{File: next(f), Format: f, FieldKey: d.key, Marker: "pcprobe-" + string(f) + "-" + d.key}
		p.addField(d, p.Marker, "pcprobe|"+string(f)+"|hierarchical", d.place)
		out = append(out, p)
	}
	combo := Probe{File: next(f), Format: f, FieldKey: KeyCombo, Marker: "pcprobe-" + string(f) + "-combo", Place: taj}
	for _, d := range defs {
		combo.addField(d, combo.Marker+"-"+d.key, "pcprobe|"+string(f)+"|combo", taj)
	}
	return append(out, combo)
}

// sidecarRow describes a video without embedded metadata whose XMP sidecar carries the markers.
func sidecarRow(f Format, variant, file string) Probe {
	p := Probe{File: file, Format: f, FieldKey: variant, Marker: "pcprobe-" + string(f) + "-" + variant, Place: machu}
	p.Sidecar = strings.TrimSuffix(file, "."+string(f)) + ".xmp"
	if variant == VariantSidecarExt {
		p.Sidecar = file + ".xmp"
	}
	for _, d := range sidecarDefs {
		p.addField(d, p.Marker+"-"+d.key, "pcprobe|"+string(f)+"|"+variant, machu)
	}
	return p
}

// addField appends the assignments of one field definition; base is the marker (or marker
// stem for location subfields) and path the hierarchical marker.
func (p *Probe) addField(d fieldDef, base, path string, place Place) {
	if p.Place == (Place{}) {
		p.Place = place
	}
	for _, t := range d.tags {
		v := base
		switch {
		case t.hierarchical:
			v = path
		case t.sub != "":
			v = base + "-" + t.sub
		}
		p.Texts = append(p.Texts, TagValue{t.tag, v})
		p.Tags = append(p.Tags, t.tag)
	}
	switch d.gps {
	case gpsEXIF:
		p.GPS = append(p.GPS,
			TagValue{"EXIF:GPSLatitude", coord(place.Lat)}, TagValue{"EXIF:GPSLatitudeRef", ref(place.Lat, "N", "S")},
			TagValue{"EXIF:GPSLongitude", coord(place.Lon)}, TagValue{"EXIF:GPSLongitudeRef", ref(place.Lon, "E", "W")})
	case gpsXMP:
		// signed values: XMP has no separate ref tags here
		p.GPS = append(p.GPS,
			TagValue{"XMP-exif:GPSLatitude", signed(place.Lat)}, TagValue{"XMP-exif:GPSLongitude", signed(place.Lon)})
	case gpsKeys:
		p.GPS = append(p.GPS, TagValue{"Keys:GPSCoordinates", signed(place.Lat) + " " + signed(place.Lon)})
	case gpsUserData:
		p.GPS = append(p.GPS, TagValue{"UserData:GPSCoordinates", signed(place.Lat) + " " + signed(place.Lon)})
	}
	if d.gps != gpsNone {
		for _, g := range p.GPS {
			if !contains(p.Tags, g.Tag) {
				p.Tags = append(p.Tags, g.Tag)
			}
		}
	}
	for _, f := range d.fixed {
		if !contains(p.Fixed, f) {
			p.Fixed = append(p.Fixed, f)
		}
	}
}

func contains[T comparable](s []T, v T) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

func signed(v float64) string { return strconv.FormatFloat(v, 'f', -1, 64) }

func coord(v float64) string {
	if v < 0 {
		v = -v
	}
	return strconv.FormatFloat(v, 'f', -1, 64)
}

func ref(v float64, pos, neg string) string {
	if v < 0 {
		return neg
	}
	return pos
}
