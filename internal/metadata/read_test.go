package metadata

import (
	"reflect"
	"testing"
)

func TestTagsMergeAllKeywordFieldsWithTheHierarchyTakingPrecedence(t *testing.T) {
	tests := []struct {
		name   string
		fields map[string]any
		want   []string
	}{
		{"tagged.jpg", map[string]any{
			"XMP-dc:Subject":             []any{"Anlass", "Geburtstag", "Familie"},
			"XMP-lr:HierarchicalSubject": "Anlass|Geburtstag",
			"IPTC:Keywords":              []any{"Familie", "Sommer"},
			"IFD0:XPKeywords":            "Sommer;Garten",
		}, []string{"Anlass/Geburtstag", "Familie", "Garten", "Sommer"}},
		{"single values and blanks", map[string]any{
			"XMP-dc:Subject": "Reise", "XMP-lr:HierarchicalSubject": "Reise|Österreich",
			"Keys:Keywords": " ", "UserData:Keywords": []any{" Video ", ""},
		}, []string{"Reise/Österreich", "Video"}},
		{"nothing", map[string]any{}, nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := Parse(tc.fields).Tags; !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestPlaceGPSAndReviewStateComeFromTheFirstFilledFieldUnknownStatesReadAsOpen(t *testing.T) {
	tests := []struct {
		name   string
		fields map[string]any
		check  func(Meta) bool
	}{
		{"xmp city beats iptc", map[string]any{"XMP-photoshop:City": "Graz", "IPTC:City": "Wien"}, func(m Meta) bool { return m.City == "Graz" }},
		{"empty xmp falls to iptc", map[string]any{"XMP-photoshop:City": " ", "IPTC:City": "Wien"}, func(m Meta) bool { return m.City == "Wien" }},
		{"ext city last", map[string]any{"XMP-iptcExt:LocationShownCity": "Linz"}, func(m Meta) bool { return m.City == "Linz" }},
		{"sublocation state country label", map[string]any{
			"IPTC:Sub-location": "Park", "IPTC:Province-State": "Stmk", "IPTC:Country-PrimaryLocationName": "AT",
			"XMP-iptcExt:LocationShownLocationName": "Park, Graz"},
			func(m Meta) bool {
				return m.Sublocation == "Park" && m.State == "Stmk" && m.Country == "AT" && m.Label == "Park, Graz"
			}},
		{"composite gps beats xmp", map[string]any{
			"Composite:GPSLatitude": 47.5, "Composite:GPSLongitude": 15.5,
			"XMP-exif:GPSLatitude": 1.0, "XMP-exif:GPSLongitude": 2.0},
			func(m Meta) bool { return m.Lat != nil && *m.Lat == 47.5 && *m.Lon == 15.5 }},
		{"xmp gps fallback", map[string]any{"XMP-exif:GPSLatitude": 1.0, "XMP-exif:GPSLongitude": 2.0},
			func(m Meta) bool { return m.Lat != nil && *m.Lat == 1 && *m.Lon == 2 }},
		{"one coordinate is no position", map[string]any{"Composite:GPSLatitude": 47.5},
			func(m Meta) bool { return m.Lat == nil && m.Lon == nil }},
		{"review done", map[string]any{"XMP-photocurator:ReviewState": "done"}, func(m Meta) bool { return m.ReviewState == "done" }},
		{"review irrelevant", map[string]any{"XMP-photocurator:ReviewState": "irrelevant"}, func(m Meta) bool { return m.ReviewState == "irrelevant" }},
		{"review unknown", map[string]any{"XMP-photocurator:ReviewState": "weird"}, func(m Meta) bool { return m.ReviewState == "open" }},
		{"review missing", map[string]any{}, func(m Meta) bool { return m.ReviewState == "open" }},
		{"exif date under ExifIFD", map[string]any{"ExifIFD:DateTimeOriginal": "2024:07:14 10:00:00", "QuickTime:CreateDate": "2024:08:20 08:00:00"},
			func(m Meta) bool {
				return m.DateTimeOriginal == "2024:07:14 10:00:00" && m.QuickTimeCreate == "2024:08:20 08:00:00"
			}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if m := Parse(tc.fields); !tc.check(m) {
				t.Errorf("unexpected %+v", m)
			}
		})
	}
}
