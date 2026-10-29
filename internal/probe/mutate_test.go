package probe

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func subjectOf(t *testing.T, path string) string {
	t.Helper()
	for _, e := range readDump(t, path) {
		if e.Group == "XMP-dc" && e.Name == "Subject" {
			return e.Value
		}
	}
	return ""
}

func mtimeOf(t *testing.T, path string) time.Time {
	t.Helper()
	st, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	return st.ModTime()
}

func TestMutate_ReplacesSubjectKeepingAndRenewingMtime(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "p")
	if code := Run([]string{"generate", "--out", dir}, os.Stderr); code != 0 {
		t.Fatal("generate failed")
	}
	keep, nw := filepath.Join(dir, MutateKeepFile), filepath.Join(dir, MutateNewFile)
	keepBefore, newBefore := mtimeOf(t, keep), mtimeOf(t, nw)
	if !keepBefore.Equal(CaptureTime()) || !newBefore.Equal(CaptureTime()) {
		t.Fatal("generated mtime is not the capture time")
	}
	for i, ver := range []string{"v2", "v3"} {
		var buf bytes.Buffer
		if code := Run([]string{"mutate", "--dir", dir}, &buf); code != 0 {
			t.Fatalf("mutate failed: %s", buf.String())
		}
		if !strings.Contains(buf.String(), "keep-"+ver) {
			t.Errorf("output lacks new marker: %s", buf.String())
		}
		if got := subjectOf(t, keep); got != mutatePrefix+"keep-"+ver {
			t.Errorf("run %d keep subject = %q", i, got)
		}
		if got := subjectOf(t, nw); got != mutatePrefix+"new-"+ver {
			t.Errorf("run %d new subject = %q", i, got)
		}
	}
	if !mtimeOf(t, keep).Equal(keepBefore) {
		t.Error("keep mtime changed")
	}
	if !mtimeOf(t, nw).After(newBefore) {
		t.Error("new mtime not renewed")
	}
}

func TestMutate_AbortsWithoutChange(t *testing.T) {
	empty := t.TempDir()
	if code := Run([]string{"mutate", "--dir", empty}, &bytes.Buffer{}); code == 0 {
		t.Error("empty dir: want failure")
	}
	if entries, _ := os.ReadDir(empty); len(entries) != 0 {
		t.Error("empty dir was modified")
	}

	dir := filepath.Join(t.TempDir(), "p")
	if code := Run([]string{"generate", "--out", dir}, os.Stderr); code != 0 {
		t.Fatal("generate failed")
	}
	keep, nw := filepath.Join(dir, MutateKeepFile), filepath.Join(dir, MutateNewFile)
	if err := WriteTags(nw, []TagValue{{"XMP-dc:Subject", "foreign"}}); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(keep)
	if code := Run([]string{"mutate", "--dir", dir}, &bytes.Buffer{}); code == 0 {
		t.Error("foreign marker: want failure")
	}
	if after, _ := os.ReadFile(keep); !bytes.Equal(before, after) {
		t.Error("keep file changed despite abort")
	}
	if got := subjectOf(t, nw); got != "foreign" {
		t.Errorf("new subject = %q", got)
	}
}
