package probe

import (
	"strings"
	"testing"
)

func TestProbes_HierarchyMarkersArePaths(t *testing.T) {
	for _, p := range Probes() {
		for _, tv := range p.Texts {
			if !strings.HasPrefix(tv.Tag, "XMP-lr:") {
				continue
			}
			kind := "hierarchical"
			if p.FieldKey == KeyCombo {
				kind = "combo"
			} else if p.Sidecar != "" {
				kind = p.FieldKey
			}
			if want := "pcprobe|" + string(p.Format) + "|" + kind; tv.Value != want {
				t.Errorf("%s: hierarchy marker %q, want path %q", p.File, tv.Value, want)
			}
		}
	}
}

func TestProbes_MarkersFollowNamingScheme(t *testing.T) {
	for _, p := range Probes() {
		want := "pcprobe-" + string(p.Format) + "-" + p.FieldKey
		if p.Marker != want {
			t.Errorf("%s: marker %q, want %q", p.File, p.Marker, want)
		}
		for _, tv := range p.Texts {
			switch {
			case strings.HasPrefix(tv.Tag, "XMP-lr:"):
				// hierarchy values are paths, covered by the path test
			case p.FieldKey == KeyCombo || p.Sidecar != "":
				if !strings.HasPrefix(tv.Value, want+"-") || strings.HasSuffix(tv.Value, "-") {
					t.Errorf("%s: %s value %q, want %s-<fieldkey>", p.File, tv.Tag, tv.Value, want)
				}
			case !strings.HasPrefix(tv.Value, want):
				t.Errorf("%s: %s value %q does not start with %q", p.File, tv.Tag, tv.Value, want)
			}
		}
	}
}

func TestProbes_MarkersUniqueAndFileNamesNeutral(t *testing.T) {
	seen := map[string]string{}
	for _, p := range Probes() {
		if prev, dup := seen[p.Marker]; dup {
			t.Errorf("marker %q used by %s and %s", p.Marker, prev, p.File)
		}
		seen[p.Marker] = p.File
		if strings.ContainsAny(p.Marker, " ") || p.Marker != strings.ToLower(p.Marker) {
			t.Errorf("marker %q must be lowercase without spaces", p.Marker)
		}
		for _, tv := range p.Texts {
			if strings.Contains(p.File, tv.Value) || strings.Contains(p.File, "pcprobe") {
				t.Errorf("file name %q contains a marker", p.File)
			}
		}
	}
}
