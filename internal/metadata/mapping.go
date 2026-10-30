package metadata

import (
	"fmt"
	"math"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Location is a place written to a file.
type Location struct {
	Lat, Lon                                 float64
	Label, Sublocation, City, State, Country string
}

// Change is one edit of a file's metadata.
type Change struct {
	AddTags, RemoveTags []string // "Parent/Child" paths
	Location            *Location
	ReviewState         string // "done", "irrelevant" or "open"; empty leaves the state alone
	// ClearLocation and ClearReviewState delete the field. A deletion runs before a Location
	// written by the same Change, so the result holds exactly that place and no older leftovers.
	ClearLocation, ClearReviewState bool `json:",omitempty"`
}

// Domain fields of a Change.
const (
	fieldTags     = "tags"
	fieldLocation = "location"
	fieldReview   = "review"
)

// ExifTool operations. List fields only ever use opAdd and opRemove: assigning a list as a
// whole would drop the keywords other tools put there.
const (
	opSet    = "="
	opAdd    = "+="
	opRemove = "-="
)

// Where a written value comes from.
const (
	srcTerm        = "term"        // a tag path segment
	srcPath        = "path"        // a tag path joined with "|"
	srcLat         = "lat"         // absolute latitude
	srcLatRef      = "lat-ref"     // N or S
	srcLon         = "lon"         // absolute longitude
	srcLonRef      = "lon-ref"     // E or W
	srcCoordinates = "coordinates" // "lat, lon"
	srcSublocation = "sublocation"
	srcLabel       = "label"
	srcCity        = "city"
	srcState       = "state"
	srcCountry     = "country"
	srcReviewState = "review-state"
	srcReviewedAt  = "reviewed-at"
)

type write struct {
	Tag, Op, Src string
}

// rule is what one domain field writes in one format.
type rule struct {
	Add    []write
	Remove []write // tag removal only
	// GPSTol is the largest coordinate difference (degrees) after a round trip.
	GPSTol float64
}

var formats = []string{"jpeg", "heic", "png", "mp4", "mov"}

const (
	imageGPSTol = 1e-6
	videoGPSTol = 1e-4
)

// table is the single mapping Format x domain field -> ExifTool tags.
var table = buildTable()

func buildTable() map[string]map[string]rule {
	t := map[string]map[string]rule{}
	for _, f := range formats {
		video := f == "mp4" || f == "mov"

		tags := rule{
			Add: []write{
				{"XMP-dc:Subject", opAdd, srcTerm},
				{"XMP-lr:HierarchicalSubject", opAdd, srcPath},
			},
			Remove: []write{
				{"XMP-dc:Subject", opRemove, srcTerm},
				{"XMP-lr:HierarchicalSubject", opRemove, srcPath},
			},
		}
		if f == "jpeg" {
			tags.Remove = append(tags.Remove, write{"IPTC:Keywords", opRemove, srcTerm})
		}

		loc := rule{GPSTol: imageGPSTol}
		if video {
			loc.GPSTol = videoGPSTol
			loc.Add = []write{
				{"Keys:GPSCoordinates", opSet, srcCoordinates},
				{"UserData:GPSCoordinates", opSet, srcCoordinates},
			}
		} else {
			loc.Add = []write{
				{"EXIF:GPSLatitude", opSet, srcLat},
				{"EXIF:GPSLatitudeRef", opSet, srcLatRef},
				{"EXIF:GPSLongitude", opSet, srcLon},
				{"EXIF:GPSLongitudeRef", opSet, srcLonRef},
			}
		}
		loc.Add = append(loc.Add,
			write{"XMP-photoshop:City", opSet, srcCity},
			write{"XMP-photoshop:State", opSet, srcState},
			write{"XMP-photoshop:Country", opSet, srcCountry},
			write{"XMP-iptcCore:Location", opSet, srcSublocation},
			write{"XMP-iptcExt:LocationShownLocationName", opSet, srcLabel},
		)

		review := rule{Add: []write{
			{"XMP-photocurator:ReviewState", opSet, srcReviewState},
			{"XMP-photocurator:ReviewedAt", opSet, srcReviewedAt},
		}}

		t[f] = map[string]rule{fieldTags: tags, fieldLocation: loc, fieldReview: review}
	}
	return t
}

// now is the clock for ReviewedAt.
var now = time.Now

func segments(tag string) []string {
	var out []string
	for _, p := range strings.Split(tag, "/") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func coordText(v float64) string { return strconv.FormatFloat(v, 'f', -1, 64) }

func ref(v float64, pos, neg string) string {
	if v < 0 {
		return neg
	}
	return pos
}

// Args builds the ExifTool arguments (without file path) that apply c to a file of format.
// old is the file's current state: it decides which parent terms are still needed.
func Args(format string, old Meta, c Change) ([]string, error) {
	rules, ok := table[format]
	if !ok {
		return nil, fmt.Errorf("unknown format %q", format)
	}
	var args []string
	emit := func(w write, value string) {
		args = append(args, "-"+w.Tag+w.Op+value)
	}

	for _, tag := range append(slices.Clone(c.AddTags), c.RemoveTags...) {
		if len(segments(tag)) == 0 {
			return nil, fmt.Errorf("empty tag %q", tag)
		}
	}

	for _, tag := range c.AddTags {
		segs := segments(tag)
		for _, w := range rules[fieldTags].Add {
			switch w.Src {
			case srcTerm:
				for _, s := range segs {
					emit(w, s)
				}
			case srcPath:
				emit(w, strings.Join(segs, "|"))
			}
		}
	}

	if len(c.RemoveTags) > 0 {
		needed := map[string]bool{}
		for _, tag := range remainingTags(old, c) {
			for _, s := range segments(tag) {
				needed[s] = true
			}
		}
		for _, tag := range c.RemoveTags {
			segs := segments(tag)
			for _, w := range rules[fieldTags].Remove {
				switch w.Src {
				case srcTerm:
					for _, s := range segs {
						if !needed[s] {
							emit(w, s)
						}
					}
				case srcPath:
					emit(w, strings.Join(segs, "|"))
				}
			}
		}
	}

	if c.ReviewState != "" && c.ClearReviewState {
		return nil, fmt.Errorf("review state both set and cleared")
	}
	// Deleting means assigning an empty value to every tag the field is written to.
	if c.ClearLocation {
		for _, w := range rules[fieldLocation].Add {
			emit(w, "")
		}
	}
	if c.ClearReviewState {
		for _, w := range rules[fieldReview].Add {
			emit(w, "")
		}
	}

	if l := c.Location; l != nil {
		for _, w := range rules[fieldLocation].Add {
			var v string
			switch w.Src {
			case srcLat:
				v = coordText(math.Abs(l.Lat))
			case srcLatRef:
				v = ref(l.Lat, "N", "S")
			case srcLon:
				v = coordText(math.Abs(l.Lon))
			case srcLonRef:
				v = ref(l.Lon, "E", "W")
			case srcCoordinates:
				v = coordText(l.Lat) + ", " + coordText(l.Lon)
			case srcSublocation:
				v = l.Sublocation
			case srcLabel:
				v = l.Label
			case srcCity:
				v = l.City
			case srcState:
				v = l.State
			case srcCountry:
				v = l.Country
			}
			if v != "" {
				emit(w, v)
			}
		}
	}

	if c.ReviewState != "" {
		for _, w := range rules[fieldReview].Add {
			switch w.Src {
			case srcReviewState:
				emit(w, c.ReviewState)
			case srcReviewedAt:
				emit(w, now().Format("2006:01:02 15:04:05-07:00"))
			}
		}
	}
	return args, nil
}

// remainingTags is old's tags with the change's additions and removals applied.
func remainingTags(old Meta, c Change) []string {
	set := map[string]bool{}
	for _, t := range old.Tags {
		set[t] = true
	}
	for _, t := range c.AddTags {
		set[strings.Join(segments(t), "/")] = true
	}
	for _, t := range c.RemoveTags {
		delete(set, strings.Join(segments(t), "/"))
	}
	out := make([]string, 0, len(set))
	for t := range set {
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}

// Expect is the Meta a read-back must show after c was applied to a file that read as old.
func Expect(old Meta, c Change) Meta {
	m := old
	m.Tags = remainingTags(old, c)
	if c.ClearLocation {
		m.Lat, m.Lon = nil, nil
		m.Label, m.Sublocation, m.City, m.State, m.Country = "", "", "", "", ""
	}
	if c.ClearReviewState {
		m.ReviewState = ReviewOpen // a file without a state reads as open
	}
	if l := c.Location; l != nil {
		lat, lon := l.Lat, l.Lon
		m.Lat, m.Lon = &lat, &lon
		for dst, src := range map[*string]string{
			&m.Label: l.Label, &m.Sublocation: l.Sublocation, &m.City: l.City, &m.State: l.State, &m.Country: l.Country,
		} {
			if src != "" {
				*dst = src
			}
		}
	}
	if c.ReviewState != "" {
		m.ReviewState = c.ReviewState
	}
	return m
}

// Matches reports whether got equals want, coordinates within the format's tolerance.
func Matches(format string, got, want Meta) bool {
	rules, ok := table[format]
	if !ok {
		return false
	}
	if !slices.Equal(got.Tags, want.Tags) {
		return false
	}
	if (got.Lat == nil) != (want.Lat == nil) || (got.Lon == nil) != (want.Lon == nil) {
		return false
	}
	tol := rules[fieldLocation].GPSTol
	if got.Lat != nil && (math.Abs(*got.Lat-*want.Lat) > tol || math.Abs(*got.Lon-*want.Lon) > tol) {
		return false
	}
	return got.Label == want.Label && got.Sublocation == want.Sublocation && got.City == want.City &&
		got.State == want.State && got.Country == want.Country && got.ReviewState == want.ReviewState
}
