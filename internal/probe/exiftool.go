package probe

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// WriteTags applies all assignments to path in ONE ExifTool call. Values travel through an
// argument file (never a shell, never argv), one per line, so a value cannot inject options;
// line breaks in values are therefore rejected. ExifTool reads argument files as UTF-8.
func WriteTags(path string, assigns []TagValue) error {
	return writeTagsWith(path, []string{"-overwrite_original"}, assigns)
}

// writeTagsWith is WriteTags with caller-chosen leading argument lines (one argument each).
func writeTagsWith(path string, head []string, assigns []TagValue) error {
	var b strings.Builder
	for _, h := range head {
		b.WriteString(h + "\n")
	}
	for _, a := range assigns {
		if strings.ContainsAny(a.Tag+a.Value, "\r\n") {
			return fmt.Errorf("line break in assignment for %s", a.Tag)
		}
		b.WriteString("-" + a.Tag + "=" + a.Value + "\n")
	}
	f, err := os.CreateTemp("", "pcprobe-args-*.txt")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err := f.WriteString(b.String()); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	out, err := exec.Command("exiftool", "-@", f.Name(), "--", path).CombinedOutput()
	// A warning means a tag was silently not written; a probe file must never be half-built.
	if err != nil || bytes.Contains(out, []byte("Warning")) || bytes.Contains(out, []byte("Error")) {
		return fmt.Errorf("exiftool write %s: %v: %s", path, err, out)
	}
	return nil
}

// Entry is one tag of a full ExifTool dump.
type Entry struct {
	Group string // family-1 group, e.g. IFD0, XMP-dc, IPTC
	Name  string
	Value string // structures and lists flattened to one string
}

var dupSuffix = regexp.MustCompile(` \(\d+\)$`)

// ReadTags returns the complete dump (-a duplicates, -G1 groups, -u unknown tags, -n raw
// values) of path, so callers can assert on every tag and not only on candidate fields.
func ReadTags(path string) ([]Entry, error) {
	out, err := exec.Command("exiftool", "-j", "-a", "-u", "-G1", "-n", "--", path).Output()
	if err != nil {
		return nil, fmt.Errorf("exiftool read %s: %w", path, err)
	}
	var docs []map[string]any
	if err := json.Unmarshal(out, &docs); err != nil || len(docs) != 1 {
		return nil, fmt.Errorf("exiftool json for %s: %v", path, err)
	}
	var entries []Entry
	for k, v := range docs[0] {
		if k == "SourceFile" {
			continue // the path we passed in, not file content
		}
		group, name, ok := strings.Cut(k, ":")
		if !ok {
			group, name = "", k
		}
		entries = append(entries, Entry{Group: group, Name: dupSuffix.ReplaceAllString(name, ""), Value: flatten(v)})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Group+":"+entries[i].Name < entries[j].Group+":"+entries[j].Name
	})
	return entries, nil
}

func flatten(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	case []any:
		parts := make([]string, len(x))
		for i, e := range x {
			parts[i] = flatten(e)
		}
		return strings.Join(parts, " ; ")
	case map[string]any:
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		parts := make([]string, len(keys))
		for i, k := range keys {
			parts[i] = flatten(x[k])
		}
		return strings.Join(parts, " ; ")
	}
	return fmt.Sprint(v)
}

// Matches reports whether e is the tag written as `tag` ("EXIF:", "IPTC:" or an XMP group
// prefix). ExifTool reports EXIF under IFD0/ExifIFD/GPS, and returns XMP structures either
// flattened (LocationShownCity) or as one parent tag (LocationShown), so names match by prefix
// in both directions.
func (e Entry) Matches(tag string) bool {
	group, name, _ := strings.Cut(tag, ":")
	if group == "EXIF" {
		switch e.Group {
		case "IFD0", "IFD1", "ExifIFD", "GPS", "InteropIFD":
		default:
			return false
		}
	} else if e.Group != group {
		return false
	}
	return strings.HasPrefix(e.Name, name) || strings.HasPrefix(name, e.Name)
}
