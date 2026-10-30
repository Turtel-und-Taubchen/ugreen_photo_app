package exif

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"photocurator/internal/metadata"
	"photocurator/internal/testsupport"
)

func newReader(t *testing.T, tags ...string) Reader {
	t.Helper()
	cfg, err := metadata.WriteConfig(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return Reader{ConfigPath: cfg, Tags: tags}
}

func TestReadBatchReturnsOneRecordPerFileKeyedByPath(t *testing.T) {
	lib := testsupport.CopyFixture(t, "library")
	good := filepath.Join(lib, "2024", "07", "tagged.jpg")
	broken := filepath.Join(lib, "2024", "08", "defekt.jpg")
	missing := filepath.Join(lib, "2024", "08", "gibt-es-nicht.jpg")
	spaced := filepath.Join(lib, "2024", "07", "-mit leer zeichen.jpg")
	if err := os.WriteFile(spaced, mustRead(t, good), 0o644); err != nil {
		t.Fatal(err)
	}
	paths := []string{good, broken, missing, spaced, good + "\n-overwrite_original"}

	unreadable := filepath.Join(lib, "2024", "07", "gesperrt.jpg")
	if err := os.WriteFile(unreadable, mustRead(t, good), 0o000); err != nil {
		t.Fatal(err)
	}
	canBlock := os.Geteuid() != 0
	if canBlock {
		paths = append(paths, unreadable)
	} else {
		t.Log("running as root: chmod 000 does not block reads, unreadable case skipped")
	}

	got, err := newReader(t, "XMP-dc:Subject", "ExifIFD:DateTimeOriginal").Read(context.Background(), paths)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(paths) {
		t.Fatalf("got %d records for %d paths: %v", len(got), len(paths), got)
	}
	for _, p := range []string{good, spaced} {
		if got[p].Err != "" || got[p].Fields["XMP-dc:Subject"] == nil {
			t.Errorf("%s: want subject without error, got %+v", p, got[p])
		}
	}
	if got[good].Fields["ExifIFD:DateTimeOriginal"] == nil {
		t.Errorf("DateTimeOriginal missing: %+v", got[good])
	}
	bad := []string{broken, missing, paths[4]}
	if canBlock {
		bad = append(bad, unreadable)
	}
	for _, p := range bad {
		if got[p].Err == "" {
			t.Errorf("%q: want an error, got %+v", p, got[p])
		}
	}
	if got[paths[4]].Err != "unsupported file name" {
		t.Errorf("line break name: %q", got[paths[4]].Err)
	}
	if strings.Contains(fmt.Sprint(got[missing].Err), broken) {
		t.Errorf("error of another file attributed to missing: %q", got[missing].Err)
	}
}

func TestReadVideoFixtureYieldsQuickTimeDateGPSAndSubject(t *testing.T) {
	lib := testsupport.CopyFixture(t, "library")
	clip := filepath.Join(lib, "2024", "08", "clip.mp4")
	got, err := newReader(t, "QuickTime:CreateDate", "Composite:GPSLatitude", "Composite:GPSLongitude", "XMP-dc:Subject").
		Read(context.Background(), []string{clip})
	if err != nil {
		t.Fatal(err)
	}
	rec := got[clip]
	if rec.Err != "" {
		t.Fatalf("error: %s", rec.Err)
	}
	for _, k := range []string{"QuickTime:CreateDate", "Composite:GPSLatitude", "Composite:GPSLongitude", "XMP-dc:Subject"} {
		if rec.Fields[k] == nil {
			t.Errorf("%s missing; have %v", k, keys(rec.Fields))
		}
	}
}

func mustRead(t *testing.T, p string) []byte {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func keys(m map[string]any) []string {
	var ks []string
	for k := range m {
		ks = append(ks, k)
	}
	slices.Sort(ks)
	return ks
}
