package writer

import (
	"bytes"
	"context"
	"math"
	"os"
	"slices"
	"strings"
	"testing"

	"photocurator/internal/metadata"
)

var (
	stephansdom = &metadata.Location{
		Lat: 48.208493, Lon: 16.373118, Label: "Stephansdom",
		Sublocation: "Innere Stadt", City: "Wien", State: "Wien", Country: "Österreich",
	}
	allFormatRels = []string{"2024/07/sample.jpg", "2024/07/sample.heic", "2024/08/screenshot.png", "2024/08/clip.mp4", "2024/08/short.mov"}
)

func runLocationJob(t *testing.T, e *env, action string, c metadata.Change, rels []string) (Job, []JobItem) {
	t.Helper()
	q := newQueue(t, e)
	ended := startQueue(t, q)
	id, err := q.Create(context.Background(), "anna", action, c, rels)
	if err != nil {
		t.Fatal(err)
	}
	<-ended
	return mustGet(t, q, id)
}

func stateOf(items []JobItem, rel string) string {
	for _, it := range items {
		if it.RelPath == rel {
			return it.State
		}
	}
	return ""
}

// forgetBackups drops every backup of rel (rows and copies), so that a backup present afterwards
// can only have been taken by the job under test.
func (e *env) forgetBackups(t *testing.T, rel string) {
	t.Helper()
	if _, err := e.p.Backups.DB.Exec(`DELETE FROM backups WHERE rel_path = ?`, rel); err != nil {
		t.Fatal(err)
	}
	for _, p := range e.backupCopiesOf(t, rel) {
		if err := os.Remove(e.p.BackupDir + "/" + p); err != nil {
			t.Fatal(err)
		}
	}
	if len(e.backupCopiesOf(t, rel)) != 0 || e.backupRows(t, rel) != 0 {
		t.Fatalf("backups of %s survived forgetBackups", rel)
	}
}

func (e *env) backupRows(t *testing.T, rel string) (n int) {
	t.Helper()
	if err := e.p.Backups.DB.QueryRow(`SELECT count(*) FROM backups WHERE rel_path = ?`, rel).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func (e *env) backupCopiesOf(t *testing.T, rel string) (out []string) {
	t.Helper()
	for _, p := range listing(t, e.p.BackupDir) {
		if strings.HasSuffix(p, "/"+rel) {
			out = append(out, p)
		}
	}
	return out
}

func (e *env) metaOf(t *testing.T, rel string) (metadata.Meta, map[string]any) {
	t.Helper()
	m, fields, err := e.p.readMetaFields(context.Background(), e.abs(rel))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	return m, fields
}

func TestSetLocationWritesMappedFieldsForEveryFormat(t *testing.T) {
	e := tagEnv(t)
	c := metadata.Change{Location: stephansdom, ClearLocation: true, LocationMode: metadata.LocationModeAll}

	job, items := runLocationJob(t, e, "location.set", c, allFormatRels)

	if job.OK != len(allFormatRels) || job.Failed != 0 || job.Skipped != 0 {
		t.Fatalf("job = %+v, items = %+v, want all %d written", job, items, len(allFormatRels))
	}
	for _, rel := range allFormatRels {
		m, fields := e.metaOf(t, rel)
		tol := 1e-6
		if slices.Contains([]string{"2024/08/clip.mp4", "2024/08/short.mov"}, rel) {
			tol = 1e-4 // video containers store coordinates with less precision
		}
		if m.Lat == nil || m.Lon == nil || math.Abs(*m.Lat-stephansdom.Lat) > tol || math.Abs(*m.Lon-stephansdom.Lon) > tol {
			t.Errorf("%s coordinates = %v, %v, want %v, %v", rel, m.Lat, m.Lon, stephansdom.Lat, stephansdom.Lon)
		}
		if m.Label != "Stephansdom" || m.Sublocation != "Innere Stadt" || m.City != "Wien" || m.State != "Wien" || m.Country != "Österreich" {
			t.Errorf("%s place = %q/%q/%q/%q/%q", rel, m.Label, m.Sublocation, m.City, m.State, m.Country)
		}
		for _, key := range []string{"XMP-photoshop:City", "XMP-photoshop:State", "XMP-photoshop:Country", "XMP-iptcCore:Location", "XMP-iptcExt:LocationShownLocationName"} {
			if fields[key] == nil {
				t.Errorf("%s lacks mapped field %s", rel, key)
			}
		}
	}
}

func TestSetLocationModeMissingSkipsLocatedFilesAndModeAllOverwrites(t *testing.T) {
	e := tagEnv(t)
	ctx := context.Background()
	const located, bare = "2024/07/sample.jpg", "2024/07/tagged.jpg"
	// Establish both starting points explicitly instead of relying on the fixtures' GPS.
	if res := e.p.WriteFile(ctx, located, metadata.Change{Location: &metadata.Location{Lat: 47.5, Lon: 8.5, City: "Altstadt"}, ClearLocation: true}); res.State == StateFailed {
		t.Fatalf("prepare located: %+v", res)
	}
	if res := e.p.WriteFile(ctx, bare, metadata.Change{ClearLocation: true}); res.State == StateFailed {
		t.Fatalf("prepare bare: %+v", res)
	}
	setMtime(t, e.abs(located), settled)
	setMtime(t, e.abs(bare), settled)
	bytesBefore := readFile(t, e.abs(located))
	fiBefore, _ := os.Stat(e.abs(located))
	e.forgetBackups(t, located)
	c := metadata.Change{Location: stephansdom, ClearLocation: true, LocationMode: metadata.LocationModeMissing}
	job, items := runLocationJob(t, e, "location.set", c, []string{located, bare})

	if job.OK != 1 || job.Skipped != 1 || job.Failed != 0 {
		t.Fatalf("job = %+v, items = %+v, want 1 written and 1 skipped", job, items)
	}
	if stateOf(items, located) != "skipped" || stateOf(items, bare) != "ok" {
		t.Errorf("items = %+v, want the located file skipped and the bare one written", items)
	}
	fiAfter, _ := os.Stat(e.abs(located))
	if !fiAfter.ModTime().Equal(fiBefore.ModTime()) || !bytes.Equal(readFile(t, e.abs(located)), bytesBefore) {
		t.Error("skipped file changed")
	}
	if n, copies := e.backupRows(t, located), e.backupCopiesOf(t, located); n != 0 || len(copies) != 0 {
		t.Errorf("skipped file got backups: %d rows, copies %v", n, copies)
	}
	if m, _ := e.metaOf(t, bare); m.City != "Wien" || m.Lat == nil {
		t.Errorf("bare file place = %+v, want Wien", m)
	}

	c.LocationMode = metadata.LocationModeAll
	job, items = runLocationJob(t, e, "location.set", c, []string{located})
	if job.OK != 1 {
		t.Fatalf("job = %+v, items = %+v, want the located file overwritten", job, items)
	}
	m, _ := e.metaOf(t, located)
	if m.City != "Wien" || m.Label != "Stephansdom" || m.Lat == nil || math.Abs(*m.Lat-stephansdom.Lat) > 1e-6 {
		t.Errorf("overwritten place = %+v, want Stephansdom in Wien", m)
	}
}

func TestSettingAPlaceWithEmptyPartsLeavesNoOlderPartsBehind(t *testing.T) {
	e := tagEnv(t)
	const rel = "2024/07/sample.jpg"
	ctx := context.Background()
	if res := e.p.WriteFile(ctx, rel, metadata.Change{Location: stephansdom, ClearLocation: true}); res.State != StateOK {
		t.Fatalf("first place: %+v", res)
	}
	bare := &metadata.Location{Lat: 1.5, Lon: 2.5, Label: "Nur ein Punkt"}

	if res := e.p.WriteFile(ctx, rel, metadata.Change{Location: bare, ClearLocation: true}); res.State != StateOK {
		t.Fatalf("second place: %+v", res)
	}

	if m, _ := e.metaOf(t, rel); m.City != "" || m.State != "" || m.Country != "" || m.Sublocation != "" || m.Label != "Nur ein Punkt" {
		t.Errorf("place = %+v, want only the label left", m)
	}
}

func TestRemoveLocationClearsCoordinatesAndPlaceFieldsAndSkipsFilesWithout(t *testing.T) {
	e := tagEnv(t)
	ctx := context.Background()
	set := metadata.Change{Location: stephansdom, ClearLocation: true}
	for _, rel := range allFormatRels {
		if res := e.p.WriteFile(ctx, rel, set); res.State != StateOK {
			t.Fatalf("prepare %s: %+v", rel, res)
		}
	}
	const bare = "2024/07/tagged.jpg"
	if res := e.p.WriteFile(ctx, bare, metadata.Change{ClearLocation: true}); res.State == StateFailed {
		t.Fatalf("prepare bare: %+v", res)
	}
	setMtime(t, e.abs(bare), settled)
	bareBytes := readFile(t, e.abs(bare))
	fiBefore, _ := os.Stat(e.abs(bare))
	e.forgetBackups(t, bare)

	job, items := runLocationJob(t, e, "location.remove", metadata.Change{ClearLocation: true}, append(slices.Clone(allFormatRels), bare))

	if job.OK != len(allFormatRels) || job.Skipped != 1 || job.Failed != 0 {
		t.Fatalf("job = %+v, items = %+v, want %d written and the bare file skipped", job, items, len(allFormatRels))
	}
	if stateOf(items, bare) != "skipped" {
		t.Errorf("bare file state = %q, want skipped", stateOf(items, bare))
	}
	for _, rel := range allFormatRels {
		if m, _ := e.metaOf(t, rel); m.HasPlace() {
			t.Errorf("%s still reads a place: %+v", rel, m)
		}
	}
	fiAfter, _ := os.Stat(e.abs(bare))
	if !fiAfter.ModTime().Equal(fiBefore.ModTime()) || !bytes.Equal(readFile(t, e.abs(bare)), bareBytes) {
		t.Error("skipped file changed")
	}
	if n, copies := e.backupRows(t, bare), e.backupCopiesOf(t, bare); n != 0 || len(copies) != 0 {
		t.Errorf("skipped file got backups: %d rows, copies %v", n, copies)
	}
}
