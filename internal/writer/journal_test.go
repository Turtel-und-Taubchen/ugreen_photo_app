package writer

import (
	"context"
	"reflect"
	"slices"
	"testing"
	"time"

	"photocurator/internal/metadata"
)

func TestJournalKeepsJobUserTimeAndOldAndNewValues(t *testing.T) {
	e := newEnv(t, true)
	q := newQueue(t, e)
	ended := startQueue(t, q)
	ctx := context.Background()
	rels := []string{sampleRel, "2024/07/tagged.jpg"} // tagged.jpg already carries foreign tags

	oldMeta := map[string]metadata.Meta{}
	for _, rel := range rels {
		m, err := e.p.readMeta(ctx, e.abs(rel))
		if err != nil {
			t.Fatal(err)
		}
		oldMeta[rel] = m
	}
	foreign := oldMeta["2024/07/tagged.jpg"].Tags
	if len(foreign) == 0 {
		t.Fatal("fixture tagged.jpg has no tags")
	}

	tagID, err := q.Create(ctx, "anna", "tags", metadata.Change{AddTags: []string{"Urlaub/Strand"}}, rels)
	if err != nil {
		t.Fatal(err)
	}
	<-ended
	job, items := mustGet(t, q, tagID)
	if job.Username != "anna" || job.ID != tagID || len(items) != 2 {
		t.Fatalf("job = %+v, %d items", job, len(items))
	}
	for _, it := range items {
		if it.State != "ok" {
			t.Fatalf("item %s: %s %s", it.RelPath, it.State, it.Reason)
		}
		if !it.WrittenAt.Equal(settled.Add(time.Hour).Truncate(time.Second)) {
			t.Errorf("%s: written_at = %v", it.RelPath, it.WrittenAt)
		}
		if it.Old.Location.Affected || it.New.Location.Affected || it.Old.ReviewState.Affected {
			t.Errorf("%s: a tag job journals other fields: %+v", it.RelPath, it)
		}
		if !it.Old.Tags.Affected || !it.New.Tags.Affected || it.New.Tags.Value == nil {
			t.Fatalf("%s: tags missing: old %+v new %+v", it.RelPath, it.Old, it.New)
		}
		if got := *it.New.Tags.Value; !slices.IsSorted(got) || !slices.Contains(got, "Urlaub/Strand") {
			t.Errorf("%s: new tags %v", it.RelPath, got)
		}
	}
	for i, rel := range rels {
		wantOld := sortedCopy(oldMeta[rel].Tags)
		wantNew := sortedCopy(append(slices.Clone(oldMeta[rel].Tags), "Urlaub/Strand"))
		wantNew = slices.Compact(wantNew)
		// A file without tags journals no old value; one with tags must journal exactly them.
		old := items[i].Old.Tags.Value
		if (old == nil && len(wantOld) > 0) || (old != nil && !slices.Equal(*old, wantOld)) {
			t.Errorf("%s: old tags = %v, want %v", rel, old, wantOld)
		}
		if nw := items[i].New.Tags.Value; nw == nil || !slices.Equal(*nw, wantNew) {
			t.Errorf("%s: new tags = %v, want %v", rel, nw, wantNew)
		}
	}

	// The location job journals what the file reads back, not the requested coordinates.
	loc := &metadata.Location{Lat: 46.62471234567, Lon: 14.30531234567, City: "Klagenfurt am Wörthersee", Country: "Österreich"}
	locID, err := q.Create(ctx, "berta", "location", metadata.Change{Location: loc}, rels)
	if err != nil {
		t.Fatal(err)
	}
	<-ended
	job, items = mustGet(t, q, locID)
	if job.Username != "berta" {
		t.Errorf("user = %q", job.Username)
	}
	for _, it := range items {
		if it.State != "ok" || it.WrittenAt.IsZero() {
			t.Fatalf("item %s: %s %s written %v", it.RelPath, it.State, it.Reason, it.WrittenAt)
		}
		if it.Old.Tags.Affected || it.New.Tags.Affected || it.New.ReviewState.Affected {
			t.Errorf("%s: a location job journals other fields: %+v", it.RelPath, it)
		}
		got, err := e.p.readMeta(ctx, e.abs(it.RelPath))
		if err != nil {
			t.Fatal(err)
		}
		want := JournalOf(got, Affected{Location: true})
		if !reflect.DeepEqual(it.New.Location, want.Location) || it.New.Location.Value == nil || it.New.Location.Value.Lat == nil {
			t.Errorf("%s: new location %+v, read back %+v", it.RelPath, it.New.Location.Value, want.Location.Value)
		}
		if v := it.New.Location.Value; v != nil && v.City != loc.City {
			t.Errorf("%s: city = %q", it.RelPath, v.City)
		}
	}
}

func sortedCopy(s []string) []string {
	c := slices.Clone(s)
	slices.Sort(c)
	return c
}
