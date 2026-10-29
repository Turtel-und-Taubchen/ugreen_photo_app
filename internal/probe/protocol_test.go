package probe

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProtocol_OneRowPerGeneratedFile(t *testing.T) {
	doc := Protocol()
	var files []string
	for _, p := range Probes() {
		files = append(files, p.File)
	}
	for _, m := range MutateFiles() {
		files = append(files, m.File)
	}
	count := func(name string) (n int, row string) {
		for _, l := range strings.Split(doc, "\n") {
			if strings.HasPrefix(l, "| ") && strings.Contains(l, name) && strings.HasSuffix(l, "| | | | | |") {
				n++
				row = l
			}
		}
		return
	}
	for _, p := range Probes() {
		n, row := count(p.File)
		if n != 1 {
			t.Errorf("%s: %d rows, want 1", p.File, n)
			continue
		}
		if p.Sidecar != "" && !strings.Contains(row, p.Sidecar) {
			t.Errorf("%s: row lacks sidecar %s", p.File, p.Sidecar)
		}
		want := p.Marker
		if strings.HasPrefix(p.FieldKey, "gps-") {
			want = p.Place.Name
		}
		for _, s := range []string{string(p.Format), p.FieldKey, want} {
			if !strings.Contains(row, "| "+s+" |") {
				t.Errorf("%s: row lacks %q: %s", p.File, s, row)
			}
		}
	}
	for _, m := range MutateFiles() {
		if n, _ := count(m.File); n != 1 {
			t.Errorf("%s: %d rows, want 1", m.File, n)
		}
	}
	dir := generated(t)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.Name() == ProtocolFile || strings.HasSuffix(e.Name(), ".xmp") {
			continue
		}
		if n, _ := count(e.Name()); n != 1 {
			t.Errorf("generated %s: %d rows, want 1", e.Name(), n)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, ProtocolFile)); err != nil {
		t.Error(err)
	}
}

func TestProtocol_TableHeaderHasObservationColumns(t *testing.T) {
	doc := Protocol()
	for _, c := range []string{"in Suche gefunden", "als Tag/Kategorie sichtbar", "in bedingtem Album nutzbar", "Ort angezeigt", "Zeit bis sichtbar"} {
		if !strings.Contains(doc, "| "+c+" |") {
			t.Errorf("table header lacks column %q", c)
		}
	}
}

func TestProtocol_ContainsDetectionGuideAndQuestions(t *testing.T) {
	doc := Protocol()
	for _, s := range []string{"## Erkennungstest", "## Prüffragen", "automatisch", "manuellem Rescan", "Verzögerung",
		"pcmut-keep.jpg", "pcmut-new.jpg", "probe mutate --dir", "PC_PRESERVE_MTIME",
		"bedingte Alben", "Schreibt UGREEN Fotos selbst Metadaten", "exiftool -a -G1",
		"probe generate --out /out", "pcprobe-jpeg-nometa", "hierarchical"} {
		if !strings.Contains(doc, s) {
			t.Errorf("protocol lacks %q", s)
		}
	}
}
