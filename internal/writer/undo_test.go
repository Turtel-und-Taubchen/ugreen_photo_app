package writer

import (
	"context"
	"database/sql"
	"errors"
	"slices"
	"sync"
	"testing"

	"photocurator/internal/metadata"
)

var klagenfurt = &metadata.Location{Lat: 46.62471234567, Lon: 14.30531234567, City: "Klagenfurt am Wörthersee", Country: "Österreich"}

// writeDirect changes a file behind the queue's back; the index is not told.
func writeDirect(t *testing.T, e *env, rel string, c metadata.Change) {
	t.Helper()
	ctx := context.Background()
	_, fields, err := e.p.readMetaFields(ctx, e.abs(rel))
	if err != nil {
		t.Fatal(err)
	}
	args, err := metadata.Args("jpeg", fields, c)
	if err != nil {
		t.Fatal(err)
	}
	if err := e.p.Exif.Write(ctx, e.abs(rel), args); err != nil {
		t.Fatal(err)
	}
}

func metaOf(t *testing.T, e *env, rel string) metadata.Meta {
	t.Helper()
	m, err := e.p.readMeta(context.Background(), e.abs(rel))
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func runJob(t *testing.T, q *Queue, ended <-chan int64, action string, c metadata.Change) int64 {
	t.Helper()
	id, err := q.Create(context.Background(), "anna", action, c, goodRels)
	if err != nil {
		t.Fatal(err)
	}
	<-ended
	return id
}

func runUndo(t *testing.T, q *Queue, ended <-chan int64, id int64) (Job, []JobItem) {
	t.Helper()
	uid, err := q.Undo(context.Background(), id, "berta")
	if err != nil {
		t.Fatal(err)
	}
	<-ended
	return mustGet(t, q, uid)
}

func TestUndoRestoresOldValuesOfEveryFile(t *testing.T) {
	e := newEnv(t, true)
	q := newQueue(t, e)
	ended := startQueue(t, q)

	before := map[string]metadata.Meta{}
	for _, rel := range goodRels {
		before[rel] = metaOf(t, e, rel)
	}
	// tagged.jpg already carries a place; the others have none.
	hadPlace := "2024/07/tagged.jpg"
	if before[hadPlace].Lat == nil {
		t.Fatalf("%s has no location", hadPlace)
	}
	if len(before["2024/07/tagged.jpg"].Tags) == 0 {
		t.Fatal("tagged.jpg has no foreign tags")
	}

	tagID := runJob(t, q, ended, "tags", metadata.Change{AddTags: []string{"Urlaub/Strand"}})
	locID := runJob(t, q, ended, "location", metadata.Change{Location: klagenfurt})

	undoLoc, items := runUndo(t, q, ended, locID)
	if undoLoc.Action != ActionUndo || undoLoc.UndoOf != locID || undoLoc.Status != JobDone || undoLoc.OK != len(goodRels) {
		t.Fatalf("undo job = %+v", undoLoc)
	}
	for _, it := range items {
		if it.State != "ok" {
			t.Fatalf("%s: %s %s", it.RelPath, it.State, it.Reason)
		}
		m := metaOf(t, e, it.RelPath)
		if it.RelPath == hadPlace {
			if m.City != before[hadPlace].City || m.Lat == nil {
				t.Errorf("%s: old place not restored: %+v", it.RelPath, m)
			}
		} else if m.Lat != nil || m.Lon != nil || m.City != "" || m.Country != "" {
			t.Errorf("%s: location left: %+v", it.RelPath, m)
		}
	}
	if src, _ := mustGet(t, q, locID); src.UndoneBy != undoLoc.ID {
		t.Errorf("UndoneBy = %d, want %d", src.UndoneBy, undoLoc.ID)
	}

	_, items = runUndo(t, q, ended, tagID)
	for _, it := range items {
		if it.State != "ok" {
			t.Fatalf("%s: %s %s", it.RelPath, it.State, it.Reason)
		}
		if got, want := sortedCopy(metaOf(t, e, it.RelPath).Tags), sortedCopy(before[it.RelPath].Tags); !slices.Equal(got, want) {
			t.Errorf("%s: tags %v, want %v", it.RelPath, got, want)
		}
	}
}

func TestUndoSkipsFileChangedSinceJobAsConflict(t *testing.T) {
	e := newEnv(t, true)
	q := newQueue(t, e)
	ended := startQueue(t, q)

	id := runJob(t, q, ended, "location", metadata.Change{Location: klagenfurt})
	changed := goodRels[1]
	other := &metadata.Location{Lat: 48.2, Lon: 16.37, City: "Wien"}
	writeDirect(t, e, changed, metadata.Change{Location: other})

	_, items := runUndo(t, q, ended, id)
	for _, it := range items {
		if it.RelPath == changed {
			if it.State != "skipped" || it.Reason != reasonConflict {
				t.Errorf("changed file: %s %q", it.State, it.Reason)
			}
			if m := metaOf(t, e, changed); m.City != "Wien" {
				t.Errorf("conflicting file was touched: %+v", m)
			}
		} else if it.State != "ok" {
			t.Errorf("%s: %s %s", it.RelPath, it.State, it.Reason)
		}
	}
}

func TestChangeToUntouchedFieldIsNoConflict(t *testing.T) {
	e := newEnv(t, true)
	q := newQueue(t, e)
	ended := startQueue(t, q)

	id := runJob(t, q, ended, "location", metadata.Change{Location: klagenfurt})
	writeDirect(t, e, goodRels[0], metadata.Change{AddTags: []string{"Spaeter"}})

	_, items := runUndo(t, q, ended, id)
	for _, it := range items {
		if it.State != "ok" {
			t.Errorf("%s: %s %s", it.RelPath, it.State, it.Reason)
		}
	}
	m := metaOf(t, e, goodRels[0])
	if m.Lat != nil || !slices.Contains(m.Tags, "Spaeter") {
		t.Errorf("after undo: %+v", m)
	}
}

func TestUndoOfAnActiveOrUnknownJobIsRefused(t *testing.T) {
	e := newEnv(t, true)
	q := newQueue(t, e) // not started: the job stays queued
	id, err := q.Create(context.Background(), "anna", "tags", metadata.Change{AddTags: []string{"A"}}, goodRels)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := q.Undo(context.Background(), id, "anna"); !errors.Is(err, ErrJobActive) {
		t.Errorf("queued job: %v", err)
	}
	if _, err := q.Undo(context.Background(), id+100, "anna"); !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("unknown job: %v", err)
	}
}

func TestSecondUndoOfTheSameJobIsRefusedAndTwoQuickCallsYieldOne(t *testing.T) {
	e := newEnv(t, true)
	q := newQueue(t, e)
	ended := startQueue(t, q)
	id := runJob(t, q, ended, "tags", metadata.Change{AddTags: []string{"A"}})

	errs := make([]error, 2)
	var wg sync.WaitGroup
	for i := range errs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, errs[i] = q.Undo(context.Background(), id, "anna")
		}()
	}
	wg.Wait()
	<-ended
	if ok := btoi(errs[0] == nil) + btoi(errs[1] == nil); ok != 1 {
		t.Fatalf("results %v, %v: want exactly one success", errs[0], errs[1])
	}
	for _, err := range errs {
		if err != nil && !errors.Is(err, ErrAlreadyUndone) {
			t.Errorf("loser: %v", err)
		}
	}
	if _, err := q.Undo(context.Background(), id, "anna"); !errors.Is(err, ErrAlreadyUndone) {
		t.Errorf("third call: %v", err)
	}
}

func btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}
