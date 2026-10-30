package metadata

import (
	"sort"
	"strconv"
	"strings"
)

// ReadTags is the single list of tags read from a file; Parse is the only place that
// interprets them. Composite GPS also covers QuickTime Keys:GPSCoordinates, which ExifTool
// derives into Composite:GPSLatitude/Longitude itself.
var ReadTags = []string{
	"XMP-dc:Subject", "XMP-lr:HierarchicalSubject", "IPTC:Keywords", "EXIF:XPKeywords",
	"Keys:Keywords", "UserData:Keywords",
	"EXIF:DateTimeOriginal", "QuickTime:CreateDate",
	"Composite:GPSLatitude", "Composite:GPSLongitude", "XMP-exif:GPSLatitude", "XMP-exif:GPSLongitude",
	"XMP-iptcCore:Location", "IPTC:Sub-location",
	"XMP-photoshop:City", "IPTC:City", "XMP-iptcExt:LocationShownCity",
	"XMP-photoshop:State", "IPTC:Province-State",
	"XMP-photoshop:Country", "IPTC:Country-PrimaryLocationName",
	"XMP-iptcExt:LocationShownLocationName",
	"XMP-photocurator:ReviewState",
}

// Meta is the domain view of one ExifTool record.
// ReviewOpen is the review state of a file that carries none.
const ReviewOpen = "open"

type Meta struct {
	Tags                              []string
	Lat, Lon                          *float64
	Sublocation, City, State, Country string
	Label                             string
	ReviewState                       string
	DateTimeOriginal, QuickTimeCreate string
}

// exifGroups are the family-1 groups ExifTool reports for what -G0 calls EXIF.
var exifGroups = []string{"EXIF", "IFD0", "ExifIFD", "IFD1", "SubIFD", "GPS", "InteropIFD"}

// lookup finds "Group:Tag" in fields; an EXIF group matches every family-1 group that
// carries EXIF tags.
func lookup(fields map[string]any, key string) (any, bool) {
	if v, ok := fields[key]; ok {
		return v, true
	}
	group, tag, ok := strings.Cut(key, ":")
	if !ok || group != "EXIF" {
		return nil, false
	}
	for _, g := range exifGroups {
		if v, ok := fields[g+":"+tag]; ok {
			return v, true
		}
	}
	return nil, false
}

// values flattens a string, number or list into trimmed, non-empty strings.
func values(v any) []string {
	var out []string
	add := func(x any) {
		var s string
		switch t := x.(type) {
		case string:
			s = t
		case float64:
			s = strconv.FormatFloat(t, 'f', -1, 64)
		default:
			return
		}
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, s)
		}
	}
	if l, ok := v.([]any); ok {
		for _, x := range l {
			add(x)
		}
	} else {
		add(v)
	}
	return out
}

func first(fields map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := lookup(fields, k); ok {
			if s := values(v); len(s) > 0 {
				return s[0]
			}
		}
	}
	return ""
}

func coord(fields map[string]any, latKey, lonKey string) (lat, lon *float64) {
	la, _ := lookup(fields, latKey)
	lo, _ := lookup(fields, lonKey)
	a, ok1 := la.(float64)
	b, ok2 := lo.(float64)
	if !ok1 || !ok2 {
		return nil, nil
	}
	return &a, &b
}

// KeywordTags are the flat keyword fields Parse reads tags from. A removed tag must vanish
// from every one of them, or the next read brings it back.
var KeywordTags = []string{"XMP-dc:Subject", "IPTC:Keywords", "EXIF:XPKeywords", "Keys:Keywords", "UserData:Keywords"}

// xpKeywords is the one keyword field that holds all entries in a single ";"-separated string.
const xpKeywords = "EXIF:XPKeywords"

// keywords is the trimmed, non-empty entries of keyword field k, in file order.
func keywords(fields map[string]any, k string) []string {
	v, ok := lookup(fields, k)
	if !ok {
		return nil
	}
	var out []string
	for _, s := range values(v) {
		items := []string{s}
		if k == xpKeywords {
			items = strings.Split(s, ";")
		}
		for _, it := range items {
			if it = strings.TrimSpace(it); it != "" {
				out = append(out, it)
			}
		}
	}
	return out
}

// Parse maps ExifTool fields (keys "Group:Tag", -n values) to Meta.
func Parse(fields map[string]any) Meta {
	var m Meta

	set := map[string]bool{}
	segments := map[string]bool{}
	if v, ok := lookup(fields, "XMP-lr:HierarchicalSubject"); ok {
		for _, h := range values(v) {
			var parts []string
			for _, p := range strings.Split(h, "|") {
				if p = strings.TrimSpace(p); p != "" {
					parts = append(parts, p)
					segments[p] = true
				}
			}
			if len(parts) > 0 {
				set[strings.Join(parts, "/")] = true
			}
		}
	}
	for _, k := range KeywordTags {
		for _, it := range keywords(fields, k) {
			if !segments[it] {
				set[it] = true
			}
		}
	}
	for t := range set {
		m.Tags = append(m.Tags, t)
	}
	sort.Strings(m.Tags)

	if m.Lat, m.Lon = coord(fields, "Composite:GPSLatitude", "Composite:GPSLongitude"); m.Lat == nil {
		m.Lat, m.Lon = coord(fields, "XMP-exif:GPSLatitude", "XMP-exif:GPSLongitude")
	}
	m.Sublocation = first(fields, "XMP-iptcCore:Location", "IPTC:Sub-location")
	m.City = first(fields, "XMP-photoshop:City", "IPTC:City", "XMP-iptcExt:LocationShownCity")
	m.State = first(fields, "XMP-photoshop:State", "IPTC:Province-State")
	m.Country = first(fields, "XMP-photoshop:Country", "IPTC:Country-PrimaryLocationName")
	m.Label = first(fields, "XMP-iptcExt:LocationShownLocationName")

	m.ReviewState = ReviewOpen
	switch s := first(fields, "XMP-photocurator:ReviewState"); s {
	case "done", "irrelevant":
		m.ReviewState = s
	}
	m.DateTimeOriginal = first(fields, "EXIF:DateTimeOriginal")
	m.QuickTimeCreate = first(fields, "QuickTime:CreateDate")
	return m
}
