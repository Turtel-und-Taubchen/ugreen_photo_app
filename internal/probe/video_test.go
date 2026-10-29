package probe

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var (
	videoKeys = []string{"xmpsubject", "hierarchical", "xmpdescription", "keys-keywords",
		"keys-description", "gps-keys", "gps-userdata", "gps-xmp", "loc-photoshop", "loc-iptccore", "loc-iptcext",
		"combo", VariantSidecarName, VariantSidecarExt}
)

func isLocationName(n string) bool {
	for _, s := range []string{"City", "State", "Country", "Location", "Province"} {
		if strings.Contains(n, s) {
			return true
		}
	}
	return false
}

func TestGenerateVideo_OneFilePerField(t *testing.T) {
	dir := generated(t)
	for _, f := range []Format{FormatMP4, FormatMOV} {
		for _, key := range videoKeys {
			n := 0
			for _, p := range videoProbes() {
				if p.Format == f && p.FieldKey == key {
					n++
					if _, err := os.Stat(filepath.Join(dir, p.File)); err != nil {
						t.Errorf("%s/%s: %v", f, key, err)
					}
				}
			}
			if n != 1 {
				t.Errorf("%s/%s: %d files, want 1", f, key, n)
			}
		}
	}
}

func TestGenerateVideo_MarkerOnlyInTargetField(t *testing.T) {
	dir := generated(t)
	for _, p := range videoProbes() {
		if p.Sidecar != "" {
			continue
		}
		dump := readDump(t, filepath.Join(dir, p.File))
		for _, tv := range p.Texts {
			if p.FieldKey == KeyCombo {
				assertMarkerOnlyIn(t, dump, tv.Value, tv.Tag)
			} else {
				assertMarkerOnlyIn(t, dump, tv.Value, p.Tags...)
			}
		}
		for _, g := range p.GPS {
			assertGPS(t, p.File, dump, g, p.Place)
		}
		for _, e := range dump {
			if e.Group == "Composite" {
				continue
			}
			isGPS := strings.HasPrefix(e.Name, "GPS")
			if len(p.GPS) == 0 && isGPS {
				t.Errorf("%s: unexpected GPS tag %s:%s", p.File, e.Group, e.Name)
			}
			if len(p.Texts) == 0 && (strings.Contains(e.Value, "pcprobe") || isLocationName(e.Name) && e.Group != "QuickTime") {
				t.Errorf("%s: GPS file carries text %s:%s", p.File, e.Group, e.Name)
			}
			if isGPS && len(p.GPS) > 0 && p.FieldKey != KeyCombo {
				listed := false
				for _, g := range p.GPS {
					listed = listed || e.Matches(g.Tag)
				}
				if !listed {
					t.Errorf("%s: foreign GPS variant %s:%s", p.File, e.Group, e.Name)
				}
			}
			if strings.HasPrefix(p.FieldKey, "loc-") && isGPS {
				t.Errorf("%s: location file has GPS %s", p.File, e.Name)
			}
		}
	}
}

func TestGenerateVideo_SidecarVariantsCarryMarkersOutsideTheVideo(t *testing.T) {
	dir := generated(t)
	for _, p := range videoProbes() {
		if p.Sidecar == "" {
			continue
		}
		ext := "." + string(p.Format)
		want := strings.TrimSuffix(p.File, ext) + ".xmp"
		if p.FieldKey == VariantSidecarExt {
			want = p.File + ".xmp"
		}
		if p.Sidecar != want {
			t.Errorf("%s/%s: sidecar %q, want %q", p.Format, p.FieldKey, p.Sidecar, want)
		}
		for _, e := range readDump(t, filepath.Join(dir, p.File)) {
			if strings.Contains(e.Value, "pcprobe") || strings.HasPrefix(e.Name, "GPS") && e.Group != "Composite" ||
				isLocationName(e.Name) && (strings.HasPrefix(e.Group, "XMP") || e.Group == "Keys" || e.Group == "UserData") {
				t.Errorf("%s: video carries %s:%s=%q", p.File, e.Group, e.Name, e.Value)
			}
		}
		side := readDump(t, filepath.Join(dir, p.Sidecar))
		for _, tv := range p.Texts {
			assertMarkerOnlyIn(t, side, tv.Value, tv.Tag)
		}
		for _, g := range p.GPS {
			assertGPS(t, p.Sidecar, side, g, machu)
		}
		info, err := os.Stat(filepath.Join(dir, p.Sidecar))
		if err != nil || !info.ModTime().Equal(CaptureTime()) {
			t.Errorf("%s: sidecar mtime %v err %v", p.Sidecar, info, err)
		}
	}
}

func TestGenerateVideo_MarkerVisibleInFrameAndDateFixed(t *testing.T) {
	dir := generated(t)
	for _, p := range videoProbes() {
		path := filepath.Join(dir, p.File)
		frame := filepath.Join(t.TempDir(), "frame.png")
		if out, err := exec.Command("ffmpeg", "-y", "-i", path, "-frames:v", "1", frame).CombinedOutput(); err != nil {
			t.Fatalf("%s: ffmpeg: %v: %s", p.File, err, out)
		}
		gray := decodeGray(t, frame)
		own := meanDeviation(gray, RenderText(p.PixelText()))
		assertForeignMarkerDeviates(t, p.File, gray, p.PixelText(), own, 0.03)
		if strings.Contains(p.File, "pcprobe") {
			t.Errorf("%s: name contains the marker", p.File)
		}
		info, err := os.Stat(path)
		if err != nil || !info.ModTime().Equal(CaptureTime()) {
			t.Errorf("%s: mtime %v err %v", p.File, info, err)
		}
		dump := readDump(t, path)
		assertDatesEqual(t, p.File, dump, videoStamp, "", "CreateDate", "ModifyDate", "TrackCreateDate", "MediaCreateDate")
	}
}
