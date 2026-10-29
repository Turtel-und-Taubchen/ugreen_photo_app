package probe

import (
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
)

var (
	genOnce sync.Once
	genDir  string
	genErr  error
	tmpRoot string
)

func TestMain(m *testing.M) {
	code := m.Run()
	if tmpRoot != "" {
		os.RemoveAll(tmpRoot)
	}
	os.Exit(code)
}

// generated runs the real command entry once per test binary and returns the output dir.
func generated(t *testing.T) string {
	t.Helper()
	genOnce.Do(func() {
		tmpRoot, genErr = os.MkdirTemp("", "pctest-")
		if genErr != nil {
			return
		}
		genDir = filepath.Join(tmpRoot, "probes")
		if code := Run([]string{"generate", "--out", genDir}, os.Stderr); code != 0 {
			genErr = os.ErrInvalid
		}
	})
	if genErr != nil {
		t.Fatalf("probe generate failed: %v", genErr)
	}
	return genDir
}

func readDump(t *testing.T, path string) []Entry {
	t.Helper()
	dump, err := ReadTags(path)
	if err != nil {
		t.Fatal(err)
	}
	return dump
}

// assertMarkerOnlyIn checks against the complete dump that value occurs in the entries of
// the target tags and in no other tag, and that it occurs at least once.
func assertMarkerOnlyIn(t *testing.T, dump []Entry, value string, targets ...string) {
	t.Helper()
	found := false
	for _, e := range dump {
		if !strings.Contains(e.Value, value) {
			continue
		}
		hit := false
		for _, tag := range targets {
			hit = hit || e.Matches(tag)
		}
		if !hit {
			t.Errorf("marker %q also found in %s:%s", value, e.Group, e.Name)
		}
		found = found || hit
	}
	if !found {
		t.Errorf("marker %q not found in %v", value, targets)
	}
}

func hasMarkerAnywhere(dump []Entry) bool {
	for _, e := range dump {
		if strings.Contains(e.Value, "pcprobe") {
			return true
		}
	}
	return false
}

// decodeGray decodes a probe picture (HEIC via vips) to 8-bit gray.
func decodeGray(t *testing.T, path string) *image.Gray {
	t.Helper()
	if strings.HasSuffix(path, ".heic") {
		conv := filepath.Join(t.TempDir(), "conv.png")
		if out, err := exec.Command("vips", "copy", path, conv).CombinedOutput(); err != nil {
			t.Fatalf("vips copy: %v: %s", err, out)
		}
		path = conv
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	img, err := decodeAny(f)
	if err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	g := image.NewGray(img.Bounds())
	for y := g.Rect.Min.Y; y < g.Rect.Max.Y; y++ {
		for x := g.Rect.Min.X; x < g.Rect.Max.X; x++ {
			g.Set(x, y, img.At(x, y))
		}
	}
	return g
}

func decodeAny(r io.ReadSeeker) (image.Image, error) {
	if img, err := jpeg.Decode(r); err == nil {
		return img, nil
	}
	if _, err := r.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	return png.Decode(r)
}

// meanDeviation is the mean absolute gray difference in 0..1 between a decoded picture and a render.
func meanDeviation(a *image.Gray, b image.Image) float64 {
	var sum float64
	r := a.Bounds()
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			bg := image.NewGray(image.Rect(0, 0, 1, 1))
			bg.Set(0, 0, b.At(x, y))
			d := float64(a.GrayAt(x, y).Y) - float64(bg.GrayAt(0, 0).Y)
			if d < 0 {
				d = -d
			}
			sum += d
		}
	}
	return sum / 255 / float64(r.Dx()*r.Dy())
}

func imageProbes() []Probe { return probesWhere(func(p Probe) bool { return !isVideo(p.Format) }) }

func videoProbes() []Probe { return probesWhere(func(p Probe) bool { return isVideo(p.Format) }) }

func isVideo(f Format) bool { return f == FormatMP4 || f == FormatMOV }

func probesWhere(keep func(Probe) bool) []Probe {
	var out []Probe
	for _, p := range Probes() {
		if keep(p) {
			out = append(out, p)
		}
	}
	return out
}

func sidecarCount() int {
	n := 0
	for _, p := range Probes() {
		if p.Sidecar != "" {
			n++
		}
	}
	return n
}

// assertGPS requires every coordinate tag of g in dump within 1e-4 of the place.
func assertGPS(t *testing.T, file string, dump []Entry, g TagValue, pl Place) {
	t.Helper()
	name := g.Tag[strings.Index(g.Tag, ":")+1:]
	if strings.HasSuffix(name, "Ref") {
		return // hemisphere letter, no coordinate
	}
	for _, e := range dump {
		if e.Group == "Composite" || !e.Matches(g.Tag) || e.Name != name {
			continue
		}
		var got []float64
		for _, s := range number.FindAllString(e.Value, -1) {
			v, _ := strconv.ParseFloat(s, 64)
			got = append(got, v)
		}
		want := []float64{pl.Lat, pl.Lon}
		switch {
		case strings.HasSuffix(name, "Latitude"):
			want = want[:1]
		case strings.HasSuffix(name, "Longitude"):
			want = want[1:]
		}
		if len(got) >= len(want) && len(want) > 0 {
			ok := true
			for i := range want {
				ok = ok && math.Abs(got[i]-want[i]) < 1e-4
			}
			if ok {
				return
			}
		}
	}
	t.Errorf("%s: %s not within 1e-4 of %v", file, g.Tag, pl)
}

var number = regexp.MustCompile(`[-+]?\d+(?:\.\d+)?`)

// assertForeignMarkerDeviates requires the picture to match its own marker text (deviation
// at most maxOwn) and to differ clearly from the same text reversed.
func assertForeignMarkerDeviates(t *testing.T, file string, gray *image.Gray, text string, own, maxOwn float64) {
	t.Helper()
	runes := []rune(text)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	foreign := meanDeviation(gray, RenderText(string(runes)))
	if own > maxOwn || foreign < 2*own || foreign < 0.01 {
		t.Errorf("%s: deviation own %.4f foreign %.4f", file, own, foreign)
	}
}

// assertDatesEqual requires each named tag to occur in dump with exactly the value want; a
// non-empty group (e.g. "EXIF") additionally restricts the match to that tag group.
func assertDatesEqual(t *testing.T, file string, dump []Entry, want, group string, names ...string) {
	t.Helper()
	for _, name := range names {
		ok := false
		for _, e := range dump {
			ok = ok || e.Name == name && e.Value == want && (group == "" || e.Matches(group+":"+name))
		}
		if !ok {
			t.Errorf("%s: %s is not %s", file, name, want)
		}
	}
}
