package metadata

import (
	"slices"
	"strings"
	"testing"
)

func TestTagArgs(t *testing.T) {
	cases := []struct {
		name        string
		fields      map[string]any
		add, remove []string
		want        []string
	}{
		{
			name: "assigning a hierarchical tag adds the path and every segment",
			add:  []string{"Anlass/Geburtstag"},
			want: []string{
				"-XMP-dc:Subject+=Anlass",
				"-XMP-dc:Subject+=Geburtstag",
				"-XMP-lr:HierarchicalSubject+=Anlass|Geburtstag",
			},
		},
		{
			name: "a file that already carries a tag in every field yields no change",
			fields: map[string]any{
				"XMP-dc:Subject":             []any{"Anlass", "Geburtstag"},
				"XMP-lr:HierarchicalSubject": "Anlass|Geburtstag",
			},
			add: []string{"Anlass/Geburtstag"},
		},
		{
			name:   "a tag present in only some fields is completed",
			fields: map[string]any{"XMP-dc:Subject": []any{"Anlass", "Sommer"}},
			add:    []string{"Anlass/Geburtstag"},
			want: []string{
				"-XMP-dc:Subject+=Geburtstag",
				"-XMP-lr:HierarchicalSubject+=Anlass|Geburtstag",
			},
		},
		{
			name: "removing a path keeps a parent another path still needs",
			fields: map[string]any{
				"XMP-dc:Subject":             []any{"Anlass", "Geburtstag", "Hochzeit"},
				"XMP-lr:HierarchicalSubject": []any{"Anlass|Geburtstag", "Anlass|Hochzeit"},
				"IPTC:Keywords":              []any{"Anlass", "Geburtstag", "Hochzeit"},
				"IFD0:XPKeywords":            "Anlass;Geburtstag;Hochzeit",
			},
			remove: []string{"Anlass/Geburtstag"},
			want: []string{
				"-XMP-lr:HierarchicalSubject-=Anlass|Geburtstag",
				"-XMP-dc:Subject-=Geburtstag",
				"-IPTC:Keywords-=Geburtstag",
				"-EXIF:XPKeywords=Anlass;Hochzeit",
			},
		},
		{
			name: "removing from XPKeywords keeps foreign entries",
			fields: map[string]any{
				"XMP-dc:Subject":  []any{"Urlaub", "Sommer"},
				"IFD0:XPKeywords": "Sommer;Urlaub;Garten",
			},
			remove: []string{"Urlaub"},
			want: []string{
				"-XMP-dc:Subject-=Urlaub",
				"-EXIF:XPKeywords=Sommer;Garten",
			},
		},
		{
			name: "removing a tag the file does not carry yields no change",
			fields: map[string]any{
				"XMP-dc:Subject":             []any{"Anlass", "Hochzeit", "Sommer"},
				"XMP-lr:HierarchicalSubject": "Anlass|Hochzeit",
				"IFD0:XPKeywords":            "Sommer",
			},
			// Anlass is only a parent of Anlass/Hochzeit here, not a tag of its own.
			remove: []string{"Anlass/Geburtstag", "Urlaub", "Anlass"},
		},
		{
			name: "removing a flat tag whose name is also a segment of a remaining path keeps it",
			fields: map[string]any{
				"XMP-dc:Subject":             []any{"Anlass", "Hochzeit"},
				"XMP-lr:HierarchicalSubject": []any{"Anlass", "Anlass|Hochzeit"},
				"IPTC:Keywords":              []any{"Anlass", "Hochzeit"},
			},
			remove: []string{"Anlass"},
			want:   []string{"-XMP-lr:HierarchicalSubject-=Anlass"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := TagArgs("jpeg", c.fields, c.add, c.remove)
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(got, c.want) {
				t.Errorf("args = %q, want %q", got, c.want)
			}
		})
	}

	t.Run("removing a tag clears it from every keyword field Parse reads", func(t *testing.T) {
		for _, format := range formats {
			fields := map[string]any{"XMP-lr:HierarchicalSubject": []any{"Reise|Italien", "Fremd|Ort"}}
			for _, k := range KeywordTags {
				if k == xpKeywords {
					fields[k] = "Fremd;Reise;Italien;Ort"
				} else {
					fields[k] = []any{"Fremd", "Reise", "Italien", "Ort"}
				}
			}
			args, err := TagArgs(format, fields, nil, []string{"Reise/Italien"})
			if err != nil {
				t.Fatal(err)
			}
			after := applyArgs(t, fields, args)
			if got, want := Parse(after).Tags, []string{"Fremd/Ort"}; !slices.Equal(got, want) {
				t.Errorf("%s: read back %q, want %q (args %q)", format, got, want, args)
			}
			for _, k := range KeywordTags {
				if got := keywords(after, k); !slices.Equal(got, []string{"Fremd", "Ort"}) {
					t.Errorf("%s: %s = %q, foreign entries must stay and the tag must go", format, k, got)
				}
			}
		}
	})
}

// applyArgs applies ExifTool tag arguments to a copy of fields the way ExifTool treats them:
// += appends to a list, -= drops matching list entries, = replaces (empty deletes).
func applyArgs(t *testing.T, fields map[string]any, args []string) map[string]any {
	t.Helper()
	out := map[string]any{}
	for k, v := range fields {
		out[k] = v
	}
	for _, a := range args {
		tag, value, ok := strings.Cut(strings.TrimPrefix(a, "-"), "=")
		if !ok {
			t.Fatalf("malformed argument %q", a)
		}
		op := tag[len(tag)-1]
		if op == '+' || op == '-' {
			tag = tag[:len(tag)-1]
		}
		var list []any
		for _, s := range values(out[tag]) {
			list = append(list, s)
		}
		switch {
		case op == '+':
			out[tag] = append(list, value)
		case op == '-':
			out[tag] = slices.DeleteFunc(list, func(x any) bool { return x == value })
		case value == "":
			delete(out, tag)
		default:
			out[tag] = value
		}
	}
	return out
}
