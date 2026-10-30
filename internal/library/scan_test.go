package library

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"sync"
	"testing"
	"time"
	_ "time/tzdata"

	"photocurator/internal/exif"
	"photocurator/internal/metadata"
	"photocurator/internal/store"
	"photocurator/internal/testsupport"
)

// fixtureMedia are the media files of testdata/library; notizen.txt is not one.
var fixtureMedia = []string{
	"2024/07/sample.heic", "2024/07/sample.jpg", "2024/07/tagged.heic", "2024/07/tagged.jpg",
	"2024/08/20240815_143005.jpg", "2024/08/IMG-20240812-WA0001.jpg", "2024/08/clip.mp4",
	"2024/08/defekt.jpg", "2024/08/screenshot.png",
}

// countingReader records which files the scanner hands to ExifTool.
type countingReader struct {
	exif.Reader
	root string
	mu   sync.Mutex
	read []string
}

func (c *countingReader) Read(ctx context.Context, paths []string) (map[string]exif.Record, error) {
	c.mu.Lock()
	for _, p := range paths {
		rel, _ := filepath.Rel(c.root, p)
		c.read = append(c.read, filepath.ToSlash(rel))
	}
	c.mu.Unlock()
	return c.Reader.Read(ctx, paths)
}

// take returns the files read since the last call, sorted.
func (c *countingReader) take() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := c.read
	c.read = nil
	sort.Strings(out)
	return out
}

type fixture struct {
	root   string
	db     *sql.DB
	reader *countingReader
	scan   *Scanner
}

func vienna(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("Europe/Vienna")
	if err != nil {
		t.Fatal(err)
	}
	return loc
}

func openScanner(t *testing.T, root, dataDir string) *fixture {
	t.Helper()
	cfg, err := metadata.WriteConfig(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	db, err := store.OpenIndexDB(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	resolved, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	r := &countingReader{Reader: exif.Reader{ConfigPath: cfg, Tags: metadata.ReadTags}, root: resolved}
	return &fixture{root: root, db: db, reader: r, scan: &Scanner{Root: root, DB: db, Reader: r, Loc: vienna(t)}}
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	return openScanner(t, testsupport.CopyFixture(t, "library"), t.TempDir())
}

func (f *fixture) run(t *testing.T) Result {
	t.Helper()
	res, err := f.scan.Run(context.Background(), nil)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	return res
}

func (f *fixture) abs(rel string) string { return filepath.Join(f.root, filepath.FromSlash(rel)) }

func copyFile(t *testing.T, src, dst string) {
	t.Helper()
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func setMtime(t *testing.T, p string, at time.Time) {
	t.Helper()
	if err := os.Chtimes(p, at, at); err != nil {
		t.Fatal(err)
	}
}

func indexedPaths(t *testing.T, db *sql.DB) []string {
	t.Helper()
	rows, err := db.Query(`SELECT rel_path FROM photos ORDER BY rel_path`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			t.Fatal(err)
		}
		out = append(out, p)
	}
	return out
}

func sorted(s ...string) []string {
	out := append([]string{}, s...)
	sort.Strings(out)
	return out
}

func without(list []string, drop ...string) []string {
	out := []string{}
	for _, s := range list {
		if !contains(drop, s) {
			out = append(out, s)
		}
	}
	return out
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

type photoRow struct {
	Folder, MediaType, Format        string
	Size, MtimeNS                    int64
	TakenAt, TakenSource             string
	Lat, Lon                         sql.NullFloat64
	Sub, City, State, Country, Label sql.NullString
	Review                           string
	ReadError                        sql.NullString
	ScannedAt                        string
	Tags                             []string
}

func loadRow(t *testing.T, db *sql.DB, rel string) (photoRow, bool) {
	t.Helper()
	var r photoRow
	var id int64
	err := db.QueryRow(`SELECT id, folder, media_type, format, size, mtime_ns, taken_at, taken_source,
		gps_lat, gps_lon, loc_sublocation, loc_city, loc_state, loc_country, loc_label,
		review_state, read_error, scanned_at FROM photos WHERE rel_path = ?`, rel).Scan(
		&id, &r.Folder, &r.MediaType, &r.Format, &r.Size, &r.MtimeNS, &r.TakenAt, &r.TakenSource,
		&r.Lat, &r.Lon, &r.Sub, &r.City, &r.State, &r.Country, &r.Label,
		&r.Review, &r.ReadError, &r.ScannedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return r, false
	}
	if err != nil {
		t.Fatal(err)
	}
	rows, err := db.Query(`SELECT tag_path FROM photo_tags WHERE photo_id = ? ORDER BY tag_path`, id)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	r.Tags = []string{}
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			t.Fatal(err)
		}
		r.Tags = append(r.Tags, tag)
	}
	return r, true
}

type wantRow struct {
	rel, mediaType, format, takenAt, source string
	tags                                    []string
	lat, lon                                float64 // 0 means no GPS
	sub, city, state, country, label        string
	review                                  string
	readError                               bool
}

// fixtureValues are the values written into the fixtures (testdata/README.md). An empty
// takenAt with source mtime means the file's modification time in Europe/Vienna.
var fixtureValues = []wantRow{
	{rel: "2024/07/tagged.jpg", mediaType: "image", format: "jpeg", takenAt: "2024-07-14T10:00:00", source: "exif",
		tags: []string{"Anlass/Geburtstag", "Familie", "Garten", "Sommer"}, lat: 47.0707, lon: 15.4395,
		sub: "Stadtpark", city: "Graz", state: "Steiermark", country: "Österreich", label: "Stadtpark, Graz", review: "done"},
	{rel: "2024/07/tagged.heic", mediaType: "image", format: "heic", takenAt: "2024-07-20T18:30:00", source: "exif",
		tags: []string{"Reise/Österreich"}, lat: 47.2692, lon: 11.4041, city: "Innsbruck", review: "irrelevant"},
	{rel: "2024/07/sample.jpg", mediaType: "image", format: "jpeg", source: "mtime", tags: []string{}, review: "open"},
	{rel: "2024/07/sample.heic", mediaType: "image", format: "heic", source: "mtime", tags: []string{}, review: "open"},
	{rel: "2024/08/clip.mp4", mediaType: "video", format: "mp4", takenAt: "2024-08-20T10:00:00", source: "quicktime",
		tags: []string{"Video"}, lat: 48.2082, lon: 16.3738, review: "open"},
	{rel: "2024/08/IMG-20240812-WA0001.jpg", mediaType: "image", format: "jpeg", takenAt: "2024-08-12T00:00:00",
		source: "filename", tags: []string{}, review: "open"},
	{rel: "2024/08/20240815_143005.jpg", mediaType: "image", format: "jpeg", takenAt: "2024-08-15T14:30:05",
		source: "filename", tags: []string{}, review: "open"},
	{rel: "2024/08/screenshot.png", mediaType: "image", format: "png", source: "mtime", tags: []string{"Notizen"}, review: "open"},
	{rel: "2024/08/defekt.jpg", mediaType: "image", format: "jpeg", source: "mtime", tags: []string{}, review: "open", readError: true},
}

func checkFixtureValues(t *testing.T, f *fixture) {
	t.Helper()
	loc := vienna(t)
	for _, w := range fixtureValues {
		got, ok := loadRow(t, f.db, w.rel)
		if !ok {
			t.Errorf("%s: not indexed", w.rel)
			continue
		}
		info, err := os.Lstat(f.abs(w.rel))
		if err != nil {
			t.Fatal(err)
		}
		takenAt := w.takenAt
		if takenAt == "" {
			takenAt = info.ModTime().In(loc).Format("2006-01-02T15:04:05")
		}
		str := func(n sql.NullString) string { return n.String }
		near := func(n sql.NullFloat64, want float64) bool {
			if want == 0 {
				return !n.Valid
			}
			return n.Valid && math.Abs(n.Float64-want) < 1e-4
		}
		if got.Folder != filepath.ToSlash(filepath.Dir(filepath.FromSlash(w.rel))) || got.MediaType != w.mediaType || got.Format != w.format ||
			got.Size != info.Size() || got.MtimeNS != info.ModTime().UnixNano() ||
			got.TakenAt != takenAt || got.TakenSource != w.source || !reflect.DeepEqual(got.Tags, w.tags) ||
			!near(got.Lat, w.lat) || !near(got.Lon, w.lon) ||
			str(got.Sub) != w.sub || str(got.City) != w.city || str(got.State) != w.state ||
			str(got.Country) != w.country || str(got.Label) != w.label || got.Review != w.review ||
			got.ReadError.Valid != w.readError || got.ScannedAt == "" {
			t.Errorf("%s:\n got %+v\nwant %+v (taken %s, size %d, mtime_ns %d)", w.rel, got, w, takenAt, info.Size(), info.ModTime().UnixNano())
		}
	}
}

func TestTheFirstScanIndexesEveryMediaFileBelowTheRootAndNothingElse(t *testing.T) {
	f := newFixture(t)
	sample := f.abs("2024/07/sample.jpg")
	for _, rel := range []string{".hidden/x.jpg", "2024/07/@eaDir/sample.jpg/thumb.jpg", "._a.jpg", "#recycle/y.jpg", "b.JpEg", "c.gif"} {
		copyFile(t, sample, f.abs(rel))
	}
	copyFile(t, f.abs("2024/08/clip.mp4"), f.abs("a.MOV"))
	if err := os.Symlink(sample, f.abs("link.jpg")); err != nil {
		t.Fatal(err)
	}

	res := f.run(t)

	want := sorted(append([]string{"a.MOV", "b.JpEg"}, fixtureMedia...)...)
	if got := indexedPaths(t, f.db); !reflect.DeepEqual(got, want) {
		t.Fatalf("indexed %v\nwant %v", got, want)
	}
	if got := f.reader.take(); !reflect.DeepEqual(got, want) {
		t.Fatalf("read %v\nwant %v", got, want)
	}
	if res.Seen != len(want) || res.Read != len(want) {
		t.Fatalf("result %+v, want Seen = Read = %d", res, len(want))
	}
	for rel, kind := range map[string][3]string{"a.MOV": {"", "video", "mov"}, "b.JpEg": {"", "image", "jpeg"}} {
		row, _ := loadRow(t, f.db, rel)
		if got := [3]string{row.Folder, row.MediaType, row.Format}; got != kind {
			t.Errorf("%s: folder/type/format %q, want %q", rel, got, kind)
		}
	}
}

func TestEveryIndexedFileCarriesPathFolderTypeSizeMtimeTakenDateWithSourceTagsGPSPlaceAndReviewState(t *testing.T) {
	f := newFixture(t)
	f.run(t)
	checkFixtureValues(t, f)
}

func TestASecondScanReadsOnlyNewOrChangedFilesAndDropsRemovedOnes(t *testing.T) {
	f := newFixture(t)
	f.run(t)
	f.reader.take()

	copyFile(t, f.abs("2024/07/sample.jpg"), f.abs("2024/08/new.jpg"))
	setMtime(t, f.abs("2024/07/sample.jpg"), time.Now().Add(-48*time.Hour))
	if err := os.Remove(f.abs("2024/08/screenshot.png")); err != nil {
		t.Fatal(err)
	}

	res := f.run(t)

	// defekt.jpg has a read error, so every scan tries it again.
	wantRead := []string{"2024/07/sample.jpg", "2024/08/defekt.jpg", "2024/08/new.jpg"}
	if got := f.reader.take(); !reflect.DeepEqual(got, wantRead) {
		t.Fatalf("read %v, want %v", got, wantRead)
	}
	want := sorted(append(without(fixtureMedia, "2024/08/screenshot.png"), "2024/08/new.jpg")...)
	if got := indexedPaths(t, f.db); !reflect.DeepEqual(got, want) {
		t.Fatalf("indexed %v\nwant %v", got, want)
	}
	if res.Removed != 1 || res.Read != 3 {
		t.Fatalf("result %+v, want Removed 1, Read 3", res)
	}
	var notes, orphans int
	f.db.QueryRow(`SELECT count(*) FROM photo_tags WHERE tag_path = 'Notizen'`).Scan(&notes)
	f.db.QueryRow(`SELECT count(*) FROM photo_tags WHERE photo_id NOT IN (SELECT id FROM photos)`).Scan(&orphans)
	if notes != 0 || orphans != 0 {
		t.Fatalf("tags of the removed file left behind: Notizen %d, orphans %d", notes, orphans)
	}
	info, _ := os.Lstat(f.abs("2024/07/sample.jpg"))
	if row, _ := loadRow(t, f.db, "2024/07/sample.jpg"); row.MtimeNS != info.ModTime().UnixNano() {
		t.Fatalf("changed file kept mtime_ns %d, want %d", row.MtimeNS, info.ModTime().UnixNano())
	}

	f.run(t)
	if got := f.reader.take(); !reflect.DeepEqual(got, []string{"2024/08/defekt.jpg"}) {
		t.Fatalf("scan without changes read %v, want only the unreadable file", got)
	}
}

func TestFilesYoungerThanTheQuietPeriodAreSkippedUntilTheySettle(t *testing.T) {
	f := newFixture(t)
	base := time.Now()
	for _, rel := range fixtureMedia {
		setMtime(t, f.abs(rel), base.Add(-2*time.Hour))
	}
	copyFile(t, f.abs("2024/07/sample.jpg"), f.abs("2024/08/fresh.jpg"))
	setMtime(t, f.abs("2024/08/fresh.jpg"), base)
	clock := base
	f.scan.Quiet = time.Hour
	f.scan.Now = func() time.Time { return clock }

	if res := f.run(t); res.Skipped != 1 {
		t.Fatalf("first scan %+v, want Skipped 1", res)
	}
	if got := indexedPaths(t, f.db); !reflect.DeepEqual(got, sorted(fixtureMedia...)) {
		t.Fatalf("indexed %v, want only the settled fixtures", got)
	}
	before, _ := loadRow(t, f.db, "2024/07/tagged.heic")
	setMtime(t, f.abs("2024/07/tagged.heic"), base.Add(-time.Minute))
	f.reader.take()

	if res := f.run(t); res.Skipped != 2 {
		t.Fatalf("second scan %+v, want Skipped 2", res)
	}
	if got := f.reader.take(); !reflect.DeepEqual(got, []string{"2024/08/defekt.jpg"}) {
		t.Fatalf("read %v while files were settling, want only the unreadable file", got)
	}
	if after, _ := loadRow(t, f.db, "2024/07/tagged.heic"); !reflect.DeepEqual(after, before) {
		t.Fatalf("row of a settling file changed:\n got %+v\nwant %+v", after, before)
	}
	if _, ok := loadRow(t, f.db, "2024/08/fresh.jpg"); ok {
		t.Fatal("a settling new file was indexed")
	}

	clock = base.Add(2 * time.Hour)
	if res := f.run(t); res.Skipped != 0 {
		t.Fatalf("third scan %+v, want Skipped 0", res)
	}
	want := []string{"2024/07/tagged.heic", "2024/08/defekt.jpg", "2024/08/fresh.jpg"}
	if got := f.reader.take(); !reflect.DeepEqual(got, want) {
		t.Fatalf("read %v after settling, want %v", got, want)
	}
	info, _ := os.Lstat(f.abs("2024/07/tagged.heic"))
	if after, _ := loadRow(t, f.db, "2024/07/tagged.heic"); after.MtimeNS != info.ModTime().UnixNano() || after.City.String != "Innsbruck" {
		t.Fatalf("settled file not re-read: %+v", after)
	}
	if _, ok := loadRow(t, f.db, "2024/08/fresh.jpg"); !ok {
		t.Fatal("settled new file not indexed")
	}
}

func TestAFileDatedFarInTheFutureIsIndexedInsteadOfWaitingForever(t *testing.T) {
	f := newFixture(t)
	now := time.Now()
	for _, rel := range fixtureMedia {
		setMtime(t, f.abs(rel), now.Add(-2*time.Hour))
	}
	const quiet = time.Hour
	copyFile(t, f.abs("2024/07/sample.jpg"), f.abs("2024/08/far.jpg"))
	setMtime(t, f.abs("2024/08/far.jpg"), now.Add(quiet*10))
	copyFile(t, f.abs("2024/07/sample.jpg"), f.abs("2024/08/near.jpg"))
	setMtime(t, f.abs("2024/08/near.jpg"), now.Add(quiet/2))
	f.scan.Quiet = quiet
	f.scan.Now = func() time.Time { return now }

	if res := f.run(t); res.Skipped != 1 {
		t.Fatalf("scan %+v, want Skipped 1", res)
	}
	if _, ok := loadRow(t, f.db, "2024/08/far.jpg"); !ok {
		t.Fatal("a file dated beyond the quiet period in the future was not indexed")
	}
	if _, ok := loadRow(t, f.db, "2024/08/near.jpg"); ok {
		t.Fatal("a file dated within the quiet period in the future was indexed")
	}
}

func TestARebuiltIndexAfterDeletingIndexDBHoldsTheSameTagsPlacesAndReviewStates(t *testing.T) {
	root := testsupport.CopyFixture(t, "library")
	dataDir := t.TempDir()
	first := openScanner(t, root, dataDir)
	first.run(t)
	checkFixtureValues(t, first)
	first.db.Close()
	for _, name := range []string{"index.db", "index.db-wal", "index.db-shm"} {
		if err := os.Remove(filepath.Join(dataDir, name)); err != nil && !errors.Is(err, fs.ErrNotExist) {
			t.Fatal(err)
		}
	}

	rebuilt := openScanner(t, root, dataDir)
	if n, _ := Count(rebuilt.db); n != 0 {
		t.Fatalf("new index holds %d rows before the scan", n)
	}
	rebuilt.run(t)
	checkFixtureValues(t, rebuilt)
	if got := indexedPaths(t, rebuilt.db); !reflect.DeepEqual(got, sorted(fixtureMedia...)) {
		t.Fatalf("rebuilt index holds %v", got)
	}
}

func TestAnUnreadableFileIsRecordedAndDoesNotStopTheScan(t *testing.T) {
	f := newFixture(t)
	locked := f.abs("2024/07/locked.jpg")
	copyFile(t, f.abs("2024/07/sample.jpg"), locked)
	wantErrors := []string{"2024/08/defekt.jpg"}
	if os.Geteuid() == 0 {
		t.Log("running as root: a chmod 000 file stays readable, only the corrupt file is checked")
	} else {
		if err := os.Chmod(locked, 0); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { os.Chmod(locked, 0o644) })
		wantErrors = sorted(append(wantErrors, "2024/07/locked.jpg")...)
	}

	res := f.run(t)

	if res.Errors != len(wantErrors) {
		t.Fatalf("result %+v, want Errors %d", res, len(wantErrors))
	}
	all := sorted(append([]string{"2024/07/locked.jpg"}, fixtureMedia...)...)
	if got := indexedPaths(t, f.db); !reflect.DeepEqual(got, all) {
		t.Fatalf("indexed %v\nwant %v", got, all)
	}
	list, total, err := Unreadable(f.db, -1)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, u := range list {
		if u.Error == "" {
			t.Errorf("%s: empty error text", u.RelPath)
		}
		got = append(got, u.RelPath)
	}
	if total != len(wantErrors) || !reflect.DeepEqual(got, wantErrors) {
		t.Fatalf("unreadable %v (total %d), want %v", got, total, wantErrors)
	}
	for _, rel := range all {
		row, _ := loadRow(t, f.db, rel)
		if row.ReadError.Valid != contains(wantErrors, rel) {
			t.Errorf("%s: read_error %v", rel, row.ReadError)
		}
		if row.ReadError.Valid && (row.TakenSource != "mtime" || len(row.Tags) != 0 || row.Lat.Valid || row.City.Valid) {
			t.Errorf("%s: unreadable file carries metadata %+v", rel, row)
		}
	}
	if first, total, _ := Unreadable(f.db, 1); len(first) != 1 || total != len(wantErrors) {
		t.Fatalf("limit 1 returned %v, total %d", first, total)
	}
}

func TestAFolderThatCannotBeReadKeepsItsFilesInTheIndex(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: chmod 000 does not make a folder unreadable")
	}
	f := newFixture(t)
	f.run(t)
	dir := f.abs("2024/08")
	if err := os.Chmod(dir, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0o755) })
	if err := os.Remove(f.abs("2024/07/sample.jpg")); err != nil {
		t.Fatal(err)
	}

	res := f.run(t)

	want := without(fixtureMedia, "2024/07/sample.jpg")
	if got := indexedPaths(t, f.db); !reflect.DeepEqual(got, sorted(want...)) {
		t.Fatalf("indexed %v\nwant %v", got, want)
	}
	if res.Removed != 1 {
		t.Fatalf("result %+v, want only the deleted file removed", res)
	}
	if row, _ := loadRow(t, f.db, "2024/08/clip.mp4"); !reflect.DeepEqual(row.Tags, []string{"Video"}) || !row.Lat.Valid {
		t.Fatalf("file in the unreadable folder lost its metadata: %+v", row)
	}
}

// treeState captures every entry below root: kind, mode, size, mtime and content hash.
func treeState(t *testing.T, root string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		state := fmt.Sprintf("%v %d %d", info.Mode(), info.Size(), info.ModTime().UnixNano())
		if info.Mode().IsRegular() {
			data, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			sum := sha256.Sum256(data)
			state += " " + hex.EncodeToString(sum[:])
		}
		out[p] = state
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestScanningLeavesEveryFileBelowTheRootUnchanged(t *testing.T) {
	f := newFixture(t)
	copyFile(t, f.abs("2024/07/sample.jpg"), f.abs(".hidden/x.jpg"))
	before := treeState(t, f.root)

	f.run(t)
	f.run(t)

	if after := treeState(t, f.root); !reflect.DeepEqual(after, before) {
		t.Fatalf("library changed by scanning:\nbefore %v\nafter  %v", before, after)
	}
}

func TestProgressCountsUpToTheNumberOfFilesToRead(t *testing.T) {
	f := newFixture(t)
	for i := range 250 {
		copyFile(t, f.abs("2024/07/sample.jpg"), f.abs(fmt.Sprintf("2024/09/bulk-%03d.jpg", i)))
	}
	var calls [][2]int
	progress := func(done, total int) { calls = append(calls, [2]int{done, total}) }

	res, err := f.scan.Run(context.Background(), progress)
	if err != nil {
		t.Fatal(err)
	}
	want := [][2]int{{200, 259}, {259, 259}}
	if !reflect.DeepEqual(calls, want) || res.Read != 259 {
		t.Fatalf("progress %v (read %d), want %v", calls, res.Read, want)
	}

	calls = nil
	if _, err := f.scan.Run(context.Background(), progress); err != nil {
		t.Fatal(err)
	}
	if want := [][2]int{{1, 1}}; !reflect.DeepEqual(calls, want) {
		t.Fatalf("progress of the rescan %v, want %v (only the unreadable file)", calls, want)
	}
}
