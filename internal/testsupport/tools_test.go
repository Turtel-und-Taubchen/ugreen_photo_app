package testsupport_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"photocurator/internal/testsupport"
)

// run executes a native tool and fails the test when it is missing or exits non-zero.
// These tests run in the Docker test stage; a missing tool is a broken image, never a skip.
func run(t *testing.T, name string, args ...string) string {
	t.Helper()
	out, err := exec.Command(name, args...).CombinedOutput()
	if err != nil {
		t.Fatalf("%s %s: %v\n%s", name, strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out))
}

func TestNativeToolsAreAvailable(t *testing.T) {
	cases := []struct {
		name string
		args []string
	}{
		{"exiftool", []string{"-ver"}},
		{"vipsthumbnail", []string{"--vips-version"}},
		{"ffmpeg", []string{"-version"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if out := run(t, c.name, c.args...); out == "" {
				t.Fatalf("%s printed no version", c.name)
			}
		})
	}
}

func headerInt(t *testing.T, field, path string) int {
	t.Helper()
	v, err := strconv.Atoi(run(t, "vipsheader", "-f", field, path))
	if err != nil {
		t.Fatalf("vipsheader %s of %s: %v", field, path, err)
	}
	return v
}

func TestLibvipsDecodesHEICFixture(t *testing.T) {
	src := testsupport.CopyFixture(t, "library/2024/07/sample.heic")
	if w, h := headerInt(t, "width", src), headerInt(t, "height", src); w <= 0 || h <= 0 {
		t.Fatalf("HEIC header size = %dx%d, want positive", w, h)
	}

	// vipsthumbnail decodes the full image, so this proves the libheif decode plugin is
	// present, not just the container parser.
	thumb := filepath.Join(t.TempDir(), "t.webp")
	run(t, "vipsthumbnail", src, "--size", "64x64", "-o", thumb)
	if _, err := os.Stat(thumb); err != nil {
		t.Fatalf("thumbnail not written: %v", err)
	}
	w, h := headerInt(t, "width", thumb), headerInt(t, "height", thumb)
	if w <= 0 || h <= 0 || w > 64 || h > 64 {
		t.Fatalf("thumbnail size = %dx%d, want within 64x64", w, h)
	}
}
