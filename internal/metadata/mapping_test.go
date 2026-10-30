package metadata

import (
	"context"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"photocurator/internal/exif"
	"photocurator/internal/testsupport"
)

// derivedByComposite lists written tags that ReadTags reads through Composite GPS.
var derivedByComposite = []string{
	"EXIF:GPSLatitude", "EXIF:GPSLatitudeRef", "EXIF:GPSLongitude", "EXIF:GPSLongitudeRef",
	"Keys:GPSCoordinates", "UserData:GPSCoordinates",
}

func tableWrites(format string) []write {
	var out []write
	for _, r := range table[format] {
		out = append(out, r.Add...)
		out = append(out, r.Remove...)
	}
	return out
}

func TestListFieldsAreNeverAssignedWhole(t *testing.T) {
	c := Change{AddTags: []string{"Anlass/Geburtstag"}, RemoveTags: []string{"Reise/Italien"}}
	for _, f := range formats {
		args, err := Args(f, Meta{Tags: []string{"Reise/Italien"}}, c)
		if err != nil {
			t.Fatal(err)
		}
		for _, w := range tableWrites(f) {
			list := strings.HasPrefix(w.Tag, "XMP-dc:Subject") || strings.HasPrefix(w.Tag, "XMP-lr:") || w.Tag == "IPTC:Keywords"
			if list && w.Op == opSet {
				t.Errorf("%s: %s is a list field mapped with =", f, w.Tag)
			}
		}
		for _, a := range args {
			for _, list := range []string{"-XMP-dc:Subject=", "-XMP-lr:HierarchicalSubject=", "-IPTC:Keywords="} {
				if strings.HasPrefix(a, list) {
					t.Errorf("%s: argument %q assigns a list field", f, a)
				}
			}
		}
	}
}

func TestEveryWrittenTagIsAlsoRead(t *testing.T) {
	for _, f := range formats {
		for _, w := range tableWrites(f) {
			// ReviewedAt is only ever written; nothing interprets it.
			if w.Tag == "XMP-photocurator:ReviewedAt" {
				continue
			}
			if !slices.Contains(ReadTags, w.Tag) && !slices.Contains(derivedByComposite, w.Tag) {
				t.Errorf("%s: %s is written but not readable", f, w.Tag)
			}
		}
	}
	for _, c := range []string{"Composite:GPSLatitude", "Composite:GPSLongitude"} {
		if !slices.Contains(ReadTags, c) {
			t.Errorf("%s missing from ReadTags", c)
		}
	}
}

func TestRemovingATagKeepsAParentAnotherTagStillNeeds(t *testing.T) {
	old := Meta{Tags: []string{"Anlass/Geburtstag", "Anlass/Hochzeit"}}
	args, err := Args("jpeg", old, Change{RemoveTags: []string{"Anlass/Geburtstag"}})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"-XMP-dc:Subject-=Geburtstag",
		"-XMP-lr:HierarchicalSubject-=Anlass|Geburtstag",
		"-IPTC:Keywords-=Geburtstag",
	}
	if !slices.Equal(args, want) {
		t.Errorf("args = %v, want %v", args, want)
	}

	args, _ = Args("jpeg", Meta{Tags: []string{"Anlass/Geburtstag"}}, Change{RemoveTags: []string{"Anlass/Geburtstag"}})
	if !slices.Contains(args, "-XMP-dc:Subject-=Anlass") {
		t.Errorf("parent nobody needs was kept: %v", args)
	}
}

func TestUnknownFormatIsAnError(t *testing.T) {
	if _, err := Args("gif", Meta{}, Change{}); err == nil {
		t.Error("unknown format accepted")
	}
}

func readFields(t *testing.T, path string, extra ...string) map[string]any {
	t.Helper()
	cfg, err := WriteConfig(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	r := exif.Reader{ConfigPath: cfg, Tags: append(slices.Clone(ReadTags), extra...)}
	recs, err := r.Read(context.Background(), []string{path})
	if err != nil {
		t.Fatal(err)
	}
	rec := recs[path]
	if rec.Err != "" {
		t.Fatalf("read %s: %s", path, rec.Err)
	}
	return rec.Fields
}

func newWriter(t *testing.T) *exif.Writer {
	t.Helper()
	cfg, err := WriteConfig(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	w := &exif.Writer{ConfigPath: cfg}
	t.Cleanup(func() { w.Close() })
	return w
}

func TestEveryFormatWritesAndReadsBackTagsPlaceAndReviewState(t *testing.T) {
	cases := []struct{ format, fixture string }{
		{"jpeg", "library/2024/07/sample.jpg"},
		{"heic", "library/2024/07/sample.heic"},
		{"png", "library/2024/08/screenshot.png"},
		{"mp4", "library/2024/08/clip.mp4"},
		{"mov", "library/2024/08/short.mov"},
	}
	change := Change{
		AddTags: []string{"Anlass/Geburtstag"},
		Location: &Location{
			Lat: 48.137154, Lon: -11.576124, Label: "Marktplatz", Sublocation: "Altstadt",
			City: "Muenchen", State: "Bayern", Country: "Deutschland",
		},
		ReviewState: "done",
	}
	for _, c := range cases {
		t.Run(c.format, func(t *testing.T) {
			path := testsupport.CopyFixture(t, c.fixture)
			w := newWriter(t)
			ctx := context.Background()
			var gpsTags []string
			for _, wr := range table[c.format][fieldLocation].Add {
				if slices.Contains(derivedByComposite, wr.Tag) {
					gpsTags = append(gpsTags, wr.Tag)
				}
			}

			old := Parse(readFields(t, path))
			hashBefore, err := w.ImageHash(ctx, path)
			if err != nil {
				t.Fatal(err)
			}
			args, err := Args(c.format, old, change)
			if err != nil {
				t.Fatal(err)
			}
			if err := w.Write(ctx, path, args); err != nil {
				t.Fatal(err)
			}

			fields := readFields(t, path, gpsTags...)
			got, want := Parse(fields), Expect(old, change)
			if !Matches(c.format, got, want) {
				t.Errorf("read back %+v, want %+v", got, want)
			}
			for _, tag := range gpsTags {
				if _, ok := lookup(fields, tag); !ok {
					t.Errorf("%s not written", tag)
				}
			}
			hashAfter, err := w.ImageHash(ctx, path)
			if err != nil {
				t.Fatal(err)
			}
			if hashBefore != hashAfter {
				t.Errorf("image data hash changed: %s -> %s", hashBefore, hashAfter)
			}
		})
	}
}

func TestForeignKeywordsSurviveAddingATag(t *testing.T) {
	path := testsupport.CopyFixture(t, filepath.ToSlash("library/2024/07/tagged.jpg"))
	w := newWriter(t)
	keys := []string{"IPTC:Keywords", "EXIF:XPKeywords", "XMP-dc:Subject"}
	before := readFields(t, path, keys...)

	old := Parse(before)
	args, err := Args("jpeg", old, Change{AddTags: []string{"Urlaub"}})
	if err != nil {
		t.Fatal(err)
	}
	if err := w.Write(context.Background(), path, args); err != nil {
		t.Fatal(err)
	}
	after := readFields(t, path, keys...)

	if !reflect.DeepEqual(valuesOf(before, "IPTC:Keywords"), []string{"Familie", "Sommer"}) {
		t.Fatalf("fixture IPTC:Keywords = %v", before["IPTC:Keywords"])
	}
	if !reflect.DeepEqual(valuesOf(before, "EXIF:XPKeywords"), []string{"Sommer;Garten"}) {
		t.Fatalf("fixture XPKeywords = %v", before["EXIF:XPKeywords"])
	}
	for _, k := range []string{"IPTC:Keywords", "EXIF:XPKeywords"} {
		if !reflect.DeepEqual(valuesOf(before, k), valuesOf(after, k)) {
			t.Errorf("%s changed: %v -> %v", k, before[k], after[k])
		}
	}
	got := valuesOf(after, "XMP-dc:Subject")
	for _, v := range append(valuesOf(before, "XMP-dc:Subject"), "Urlaub") {
		if !slices.Contains(got, v) {
			t.Errorf("XMP-dc:Subject lost %q: %v", v, got)
		}
	}
}

func valuesOf(fields map[string]any, key string) []string {
	v, _ := lookup(fields, key)
	return values(v)
}
