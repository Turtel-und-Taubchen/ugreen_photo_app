package probe

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var (
	commonKeys = []string{"xmpsubject", "hierarchical", "xmpdescription", "exifdescription",
		"gps-exif", "gps-xmp", "loc-photoshop", "loc-iptccore", "loc-iptcext", "combo"}
	jpegOnlyKeys = []string{"iptckeywords", "xpkeywords", "iptccaption", "loc-iptc"}
)

func wantKeys(f Format) []string {
	keys := append([]string(nil), commonKeys...)
	if f == FormatJPEG {
		keys = append(keys, jpegOnlyKeys...)
	}
	return keys
}

func TestGenerate_OneFilePerFormatAndField(t *testing.T) {
	dir := generated(t)
	// generate writes exactly one file per format and candidate field
	for _, f := range []Format{FormatJPEG, FormatHEIC, FormatPNG} {
		for _, key := range wantKeys(f) {
			var matches []Probe
			for _, p := range Probes() {
				if p.Format == f && p.FieldKey == key {
					matches = append(matches, p)
				}
			}
			if len(matches) != 1 {
				t.Fatalf("%s/%s: %d probes, want 1", f, key, len(matches))
			}
			p := matches[0]
			if _, err := os.Stat(filepath.Join(dir, p.File)); err != nil {
				t.Errorf("%s/%s: %v", f, key, err)
			}
		}
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if want := len(Probes()) + sidecarCount() + len(MutateFiles()) + 1; len(entries) != want {
		t.Errorf("%d files, want %d", len(entries), want)
	}
}

func TestGenerate_MarkerOnlyInTargetField(t *testing.T) {
	dir := generated(t)
	for _, p := range imageProbes() {
		if len(p.Texts) == 0 || p.FieldKey == KeyCombo {
			continue
		}
		dump := readDump(t, filepath.Join(dir, p.File))
		for _, tv := range p.Texts {
			assertMarkerOnlyIn(t, dump, tv.Value, p.Tags...)
		}
	}
}

func TestGenerate_GPSOnlyAndLocationTextOnlyFiles(t *testing.T) {
	dir := generated(t)
	for _, p := range imageProbes() {
		if p.FieldKey == KeyCombo || p.FieldKey == KeyNoMeta {
			continue
		}
		dump := readDump(t, filepath.Join(dir, p.File))
		if len(p.GPS) == 0 {
			for _, e := range dump {
				if strings.HasPrefix(e.Name, "GPS") {
					t.Errorf("%s: unexpected GPS tag %s:%s", p.File, e.Group, e.Name)
				}
			}
			continue
		}
		if len(p.Texts) != 0 {
			t.Errorf("%s: GPS file carries text markers", p.File)
		}
		for _, g := range p.GPS {
			assertGPS(t, p.File, dump, g, p.Place)
		}
		// the other GPS variant must be absent
		for _, e := range dump {
			if e.Name == "GPSLatitude" && e.Group != "Composite" && !e.Matches(p.GPS[0].Tag) {
				t.Errorf("%s: foreign GPS variant %s", p.File, e.Group)
			}
		}
		for _, e := range dump {
			if strings.Contains(e.Value, "pcprobe") {
				t.Errorf("%s: text marker in %s:%s", p.File, e.Group, e.Name)
			}
		}
	}
}

func TestGenerate_ComboAndWhatsAppLikeFile(t *testing.T) {
	dir := generated(t)
	for _, p := range imageProbes() {
		switch p.FieldKey {
		case KeyCombo:
			dump := readDump(t, filepath.Join(dir, p.File))
			for _, tv := range p.Texts {
				if !strings.HasPrefix(tv.Value, "pcprobe-"+string(p.Format)+"-combo-") && !strings.HasPrefix(tv.Value, "pcprobe|") {
					t.Errorf("%s: combo marker %q", p.File, tv.Value)
				}
				assertMarkerOnlyIn(t, dump, tv.Value, tv.Tag)
			}
			if len(p.GPS) != 6 || p.Place.Name != "Taj Mahal" {
				t.Errorf("%s: combo GPS %v place %q", p.File, p.GPS, p.Place.Name)
			}
			for _, gv := range p.GPS {
				assertGPS(t, p.File, dump, gv, p.Place)
			}
		case KeyNoMeta:
			dump := readDump(t, filepath.Join(dir, p.File))
			if hasMarkerAnywhere(dump) {
				t.Errorf("%s: carries a marker in metadata", p.File)
			}
			for _, e := range dump {
				if strings.HasPrefix(e.Group, "XMP") || strings.HasPrefix(e.Group, "IPTC") ||
					strings.HasPrefix(e.Group, "IFD") || e.Group == "ExifIFD" || e.Group == "GPS" {
					t.Errorf("%s: metadata tag %s:%s remains", p.File, e.Group, e.Name)
				}
			}
		}
	}
	if p := Probes()[len(Probes())-1]; p.File != WhatsAppFile {
		t.Errorf("last probe %q, want the WhatsApp-like file", p.File)
	}
}

func TestGenerate_MarkerVisibleInPixelsDateFixedNameNeutral(t *testing.T) {
	dir := generated(t)
	for _, p := range imageProbes() {
		path := filepath.Join(dir, p.File)
		gray := decodeGray(t, path)
		own := meanDeviation(gray, RenderText(p.PixelText()))
		assertForeignMarkerDeviates(t, p.File, gray, p.PixelText(), own, 0.02)

		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if !info.ModTime().Equal(CaptureTime()) {
			t.Errorf("%s: mtime %v", p.File, info.ModTime())
		}
		if strings.Contains(p.File, "pcprobe") {
			t.Errorf("%s: name contains the marker", p.File)
		}
		if p.FieldKey == KeyNoMeta {
			continue
		}
		dump := readDump(t, path)
		assertDatesEqual(t, p.File, dump, captureStamp, "EXIF", "DateTimeOriginal", "CreateDate")
	}
}

func TestGenerate_RefusesNonEmptyOutputDir(t *testing.T) {
	dir := t.TempDir()
	keep := filepath.Join(dir, "keep.txt")
	if err := os.WriteFile(keep, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stderr strings.Builder
	if code := Run([]string{"generate", "--out", dir}, &stderr); code == 0 {
		t.Fatal("generate accepted a non-empty directory")
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 1 {
		t.Errorf("directory was modified: %d entries", len(entries))
	}
}
