package writer

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"testing"
	"time"

	"photocurator/internal/exif"
	"photocurator/internal/library"
	"photocurator/internal/metadata"
	"photocurator/internal/store"
	"photocurator/internal/testsupport"
)

const sampleRel = "2024/07/sample.jpg"

// settled is the mtime given to the file under test; the pipeline's clock is an hour later.
var settled = time.Date(2024, 7, 1, 12, 0, 0, 0, time.UTC)

type env struct {
	root string
	db   *sql.DB
	p    *Pipeline
}

func newEnv(t *testing.T, preserveMtime bool) *env {
	t.Helper()
	cfg, err := metadata.WriteConfig(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	root := testsupport.CopyFixture(t, "library")
	db, err := store.OpenIndexDB(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	reader := exif.Reader{ConfigPath: cfg, Tags: metadata.ReadTags}
	w := &exif.Writer{ConfigPath: cfg, PreserveMtime: preserveMtime}
	t.Cleanup(func() { w.Close() })
	e := &env{root: root, db: db, p: &Pipeline{
		Root: root, BackupDir: t.TempDir(), Exif: w, Reader: reader,
		Index: &library.Scanner{Root: root, DB: db, Reader: reader},
		Quiet: 5 * time.Minute, Now: func() time.Time { return settled.Add(time.Hour) },
	}}
	setMtime(t, e.abs(sampleRel), settled)
	return e
}

func (e *env) abs(rel string) string { return filepath.Join(e.root, filepath.FromSlash(rel)) }

func setMtime(t *testing.T, p string, at time.Time) {
	t.Helper()
	if err := os.Chtimes(p, at, at); err != nil {
		t.Fatal(err)
	}
}

func statOf(t *testing.T, p string) *syscall.Stat_t {
	t.Helper()
	fi, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	return fi.Sys().(*syscall.Stat_t)
}

func readFile(t *testing.T, p string) []byte {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func listing(t *testing.T, root string) []string {
	t.Helper()
	var out []string
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		r, _ := filepath.Rel(root, p)
		out = append(out, filepath.ToSlash(r))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func (e *env) assertNoInflightCopies(t *testing.T) {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(e.p.BackupDir, inflightDir))
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("in-flight folder not empty: %v", listing(t, filepath.Join(e.p.BackupDir, inflightDir)))
	}
}

func TestWriteKeepsInodeOwnerModeAndLeavesNoExtraFiles(t *testing.T) {
	e := newEnv(t, true)
	before := statOf(t, e.abs(sampleRel))
	filesBefore := listing(t, e.root)

	res := e.p.WriteFile(context.Background(), sampleRel, metadata.Change{AddTags: []string{"Urlaub/Strand"}})
	if res.State != StateOK {
		t.Fatalf("result = %+v, want ok", res)
	}

	after := statOf(t, e.abs(sampleRel))
	if after.Ino != before.Ino || after.Uid != before.Uid || after.Gid != before.Gid || after.Mode != before.Mode {
		t.Errorf("stat changed: before ino=%d uid=%d gid=%d mode=%o, after ino=%d uid=%d gid=%d mode=%o",
			before.Ino, before.Uid, before.Gid, before.Mode, after.Ino, after.Uid, after.Gid, after.Mode)
	}
	if got := listing(t, e.root); !slices.Equal(got, filesBefore) {
		t.Errorf("library listing changed:\nbefore %v\nafter  %v", filesBefore, got)
	}
	e.assertNoInflightCopies(t)
}

func TestReadBackMismatchRestoresFileAndReportsFailure(t *testing.T) {
	e := newEnv(t, true)
	original := readFile(t, e.abs(sampleRel))
	ino := statOf(t, e.abs(sampleRel)).Ino
	e.p.AfterWrite = func(abs string) error {
		return e.p.Exif.Write(context.Background(), abs, []string{"-XMP-dc:Subject+=Fremd"})
	}

	res := e.p.WriteFile(context.Background(), sampleRel, metadata.Change{AddTags: []string{"Urlaub"}})
	if res.State != StateFailed || !strings.Contains(res.Reason, "Read-back") {
		t.Errorf("result = %+v, want failed with read-back reason", res)
	}
	if !bytes.Equal(readFile(t, e.abs(sampleRel)), original) {
		t.Error("file bytes differ from before the write")
	}
	if got := statOf(t, e.abs(sampleRel)).Ino; got != ino {
		t.Errorf("inode %d, want %d: restore replaced the file", got, ino)
	}
	e.assertNoInflightCopies(t)
}

func TestMtimeFollowsPreserveMtimeSetting(t *testing.T) {
	for _, preserve := range []bool{true, false} {
		e := newEnv(t, preserve)
		res := e.p.WriteFile(context.Background(), sampleRel, metadata.Change{ReviewState: "done"})
		if res.State != StateOK {
			t.Fatalf("preserve=%v: result = %+v, want ok", preserve, res)
		}
		fi, err := os.Stat(e.abs(sampleRel))
		if err != nil {
			t.Fatal(err)
		}
		if kept := fi.ModTime().Equal(settled); kept != preserve {
			t.Errorf("preserve=%v: mtime %v, before %v", preserve, fi.ModTime(), settled)
		}
	}
}

func TestFilesInQuietPeriodOrOutsidePhotosAreSkipped(t *testing.T) {
	e := newEnv(t, true)
	now := settled.Add(time.Hour)
	e.p.Backups = newBackups(t, e, &now)
	ctx := context.Background()
	change := metadata.Change{AddTags: []string{"Urlaub"}}

	setMtime(t, e.abs(sampleRel), e.p.Now().Add(-time.Minute))
	fresh := readFile(t, e.abs(sampleRel))
	if res := e.p.WriteFile(ctx, sampleRel, change); res.State != StateSkipped || res.Reason != "Ruhezeit" {
		t.Errorf("fresh file: result = %+v, want skipped Ruhezeit", res)
	}
	if !bytes.Equal(readFile(t, e.abs(sampleRel)), fresh) {
		t.Error("fresh file was changed")
	}

	outside := filepath.Join(filepath.Dir(e.root), "x.jpg")
	if err := os.WriteFile(outside, readFile(t, e.abs("2024/07/tagged.jpg")), 0o644); err != nil {
		t.Fatal(err)
	}
	setMtime(t, outside, settled)
	outsideBytes := readFile(t, outside)
	if err := os.Symlink(outside, e.abs("2024/07/link.jpg")); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{"../x.jpg", "2024/07/link.jpg"} {
		if res := e.p.WriteFile(ctx, rel, change); res.State != StateSkipped || res.Reason != reasonOutside {
			t.Errorf("%s: result = %+v, want skipped %q", rel, res, reasonOutside)
		}
	}
	if !bytes.Equal(readFile(t, outside), outsideBytes) {
		t.Error("file outside the library was changed")
	}
	e.assertNoInflightCopies(t)
	if got := listing(t, e.p.BackupDir); len(got) > 2 || (len(got) == 2 && got[1] != inflightDir) {
		t.Errorf("a skipped file got a backup: %v", got)
	}
}

func TestIndexShowsWrittenValuesRightAfterTheWrite(t *testing.T) {
	e := newEnv(t, true)
	ctx := context.Background()
	if _, err := e.p.Index.Run(ctx, nil); err != nil {
		t.Fatal(err)
	}

	res := e.p.WriteFile(ctx, sampleRel, metadata.Change{
		AddTags:     []string{"Urlaub/Österreich"},
		Location:    &metadata.Location{Lat: 46.6247, Lon: 14.3053, City: "Klagenfurt am Wörthersee", Country: "Österreich"},
		ReviewState: "done",
	})
	if res.State != StateOK {
		t.Fatalf("result = %+v, want ok", res)
	}

	var city, country, review string
	var lat float64
	err := e.db.QueryRow(`SELECT loc_city, loc_country, review_state, gps_lat FROM photos WHERE rel_path = ?`, sampleRel).
		Scan(&city, &country, &review, &lat)
	if err != nil {
		t.Fatal(err)
	}
	if city != "Klagenfurt am Wörthersee" || country != "Österreich" || review != "done" || lat < 46.62 || lat > 46.63 {
		t.Errorf("index row city=%q country=%q review=%q lat=%v", city, country, review, lat)
	}
	rows, err := e.db.Query(`SELECT t.tag_path FROM photo_tags t JOIN photos p ON p.id = t.photo_id WHERE p.rel_path = ?`, sampleRel)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var tags []string
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			t.Fatal(err)
		}
		tags = append(tags, tag)
	}
	if !slices.Contains(tags, "Urlaub/Österreich") {
		t.Errorf("indexed tags %v lack Urlaub/Österreich", tags)
	}
}

func TestRecoverRestoresAHalfWrittenFileFromItsInflightCopy(t *testing.T) {
	e := newEnv(t, true)
	ctx := context.Background()
	original := readFile(t, e.abs(sampleRel))
	ino := statOf(t, e.abs(sampleRel)).Ino
	e.p.AfterWrite = func(abs string) error {
		if err := os.Truncate(abs, int64(len(original)/2)); err != nil {
			return err
		}
		return errors.New("simulated crash")
	}

	if res := e.p.WriteFile(ctx, sampleRel, metadata.Change{AddTags: []string{"Urlaub"}}); res.State != StateFailed {
		t.Fatalf("result = %+v, want failed", res)
	}
	if got := len(readFile(t, e.abs(sampleRel))); got != len(original)/2 {
		t.Fatalf("file has %d bytes after the crash, want %d", got, len(original)/2)
	}

	e.p.AfterWrite = nil
	restored, err := e.p.Recover(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(restored, []string{sampleRel}) {
		t.Errorf("restored %v, want [%s]", restored, sampleRel)
	}
	if !bytes.Equal(readFile(t, e.abs(sampleRel)), original) {
		t.Error("file bytes differ from the original after Recover")
	}
	if got := statOf(t, e.abs(sampleRel)).Ino; got != ino {
		t.Errorf("inode %d, want %d: Recover replaced the file", got, ino)
	}
	e.assertNoInflightCopies(t)
}

func TestReadOnlyFileFailsWithoutLeavingAPreCopyAndDoesNotBlockLaterWrites(t *testing.T) {
	e := newEnv(t, true)
	ctx := context.Background()
	change := metadata.Change{AddTags: []string{"Urlaub"}}
	original := readFile(t, e.abs(sampleRel))
	// The tests run unprivileged (see the test stage of the Dockerfile), so 0444 really blocks ExifTool.
	if err := os.Chmod(e.abs(sampleRel), 0o444); err != nil {
		t.Fatal(err)
	}

	res := e.p.WriteFile(ctx, sampleRel, change)
	if res.State != StateFailed || res.Reason == "" || strings.Contains(res.Reason, "Wiederherstellung") {
		t.Errorf("result = %+v, want failed with a reason and no pending restore", res)
	}
	if !bytes.Equal(readFile(t, e.abs(sampleRel)), original) {
		t.Error("file bytes differ from before the write")
	}
	e.assertNoInflightCopies(t)

	if err := os.Chmod(e.abs(sampleRel), 0o644); err != nil {
		t.Fatal(err)
	}
	if res := e.p.WriteFile(ctx, sampleRel, change); res.State != StateOK {
		t.Errorf("second write: result = %+v, want ok", res)
	}
}

func TestRecoverDropsTheCopyOfAnUnchangedReadOnlyOriginal(t *testing.T) {
	e := newEnv(t, true)
	ctx := context.Background()
	original := readFile(t, e.abs(sampleRel))
	e.p.AfterWrite = func(abs string) error {
		// The crash hit before ExifTool changed anything: the original still equals the copy.
		if err := os.WriteFile(abs, original, 0o644); err != nil {
			return err
		}
		return errors.New("simulated crash")
	}
	if res := e.p.WriteFile(ctx, sampleRel, metadata.Change{AddTags: []string{"Urlaub"}}); res.State != StateFailed {
		t.Fatalf("result = %+v, want failed", res)
	}
	e.p.AfterWrite = nil
	if err := os.Chmod(e.abs(sampleRel), 0o444); err != nil {
		t.Fatal(err)
	}

	restored, err := e.p.Recover(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(restored) != 0 {
		t.Errorf("restored %v, want none: the original was never changed", restored)
	}
	if !bytes.Equal(readFile(t, e.abs(sampleRel)), original) {
		t.Error("file bytes changed")
	}
	e.assertNoInflightCopies(t)
}

func TestHashMismatchRestoresFileEvenWhenTheReadBackMatches(t *testing.T) {
	e := newEnv(t, true)
	original := readFile(t, e.abs(sampleRel))
	ino := statOf(t, e.abs(sampleRel)).Ino
	e.p.AfterWrite = func(abs string) error {
		b := readFile(t, abs)
		sos := bytes.LastIndex(b, []byte{0xFF, 0xDA})
		if sos < 0 {
			t.Error("no start-of-scan marker in the written file")
			return nil
		}
		start := sos + 2 + int(b[sos+2])<<8 + int(b[sos+3])
		end := len(b) - 2 // keep the end-of-image marker
		if start >= end {
			t.Error("no image data after the start-of-scan header")
			return nil
		}
		// Flip a byte that cannot form or touch a marker; metadata sits before the scan.
		for i := start + (end-start)/2; i < end; i++ {
			if b[i] != 0xFF && b[i] != 0xFE && b[i-1] != 0xFF {
				b[i] ^= 0x01
				return os.WriteFile(abs, b, 0o644)
			}
		}
		t.Error("no byte to flip in the image data")
		return nil
	}

	res := e.p.WriteFile(context.Background(), sampleRel, metadata.Change{AddTags: []string{"Urlaub"}})
	if res.State != StateFailed || !strings.Contains(res.Reason, "Bilddaten verändert") {
		t.Errorf("result = %+v, want failed with image data reason", res)
	}
	if !bytes.Equal(readFile(t, e.abs(sampleRel)), original) {
		t.Error("file bytes differ from before the write")
	}
	if got := statOf(t, e.abs(sampleRel)).Ino; got != ino {
		t.Errorf("inode %d, want %d: restore replaced the file", got, ino)
	}
	e.assertNoInflightCopies(t)
}
