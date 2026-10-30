package metadata

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"photocurator/internal/testsupport"
)

func TestPhotocuratorNamespaceRoundTripsTheReviewState(t *testing.T) {
	cfg, err := WriteConfig(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(testsupport.CopyFixture(t, "library/2024/07/sample.jpg"))
	if out, err := exec.Command("exiftool", "-config", cfg, "-overwrite_original",
		"-XMP-photocurator:ReviewState=done", file).CombinedOutput(); err != nil {
		t.Fatalf("write: %v: %s", err, out)
	}
	out, err := exec.Command("exiftool", "-config", cfg, "-s3", "-XMP-photocurator:ReviewState", file).Output()
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(string(out)); got != "done" {
		t.Fatalf("ReviewState = %q, want done", got)
	}
}
