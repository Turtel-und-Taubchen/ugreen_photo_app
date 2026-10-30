package writer

import (
	"bytes"
	"context"
	"os"
	"slices"
	"sort"
	"testing"
	"time"

	"photocurator/internal/library"
	"photocurator/internal/metadata"
)

// tagEnv is a pipeline whose fixture files are all past the quiet period, with backups on.
func tagEnv(t *testing.T) *env {
	t.Helper()
	e := newEnv(t, true)
	for _, rel := range listing(t, e.root) {
		if fi, err := os.Stat(e.abs(rel)); err == nil && fi.Mode().IsRegular() {
			setMtime(t, e.abs(rel), settled)
		}
	}
	now := time.Date(2024, 9, 10, 15, 30, 0, 0, time.Local)
	e.p.Backups = newBackups(t, e, &now)
	return e
}

func (e *env) tagsOf(t *testing.T, rel string) []string {
	t.Helper()
	m, err := e.p.readMeta(context.Background(), e.abs(rel))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	return m.Tags
}

func runTagJob(t *testing.T, e *env, c metadata.Change, rels []string) (Job, []JobItem) {
	t.Helper()
	q := newQueue(t, e)
	ended := startQueue(t, q)
	id, err := q.Create(context.Background(), "anna", "tags", c, rels)
	if err != nil {
		t.Fatal(err)
	}
	<-ended
	return mustGet(t, q, id)
}

func TestTaggingASelectionWritesAndReadsBackTheTagsOnJpegHeicPngMp4AndMov(t *testing.T) {
	e := tagEnv(t)
	rels := []string{"2024/07/sample.jpg", "2024/07/sample.heic", "2024/08/screenshot.png", "2024/08/clip.mp4", "2024/08/short.mov"}

	ctx := context.Background()
	before := map[string][]string{}
	for _, rel := range rels {
		before[rel] = e.tagsOf(t, rel)
	}

	job, items := runTagJob(t, e, metadata.Change{AddTags: []string{"Anlass/Hochzeit"}}, rels)

	if job.OK != len(rels) || job.Failed != 0 || job.Skipped != 0 {
		t.Fatalf("job = %+v, items = %+v, want all %d written", job, items, len(rels))
	}
	for _, rel := range rels {
		m, fields, err := e.p.readMetaFields(ctx, e.abs(rel))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		if !slices.Contains(m.Tags, "Anlass/Hochzeit") {
			t.Errorf("%s reads back tags %v, want Anlass/Hochzeit", rel, m.Tags)
		}
		for _, old := range before[rel] {
			if !slices.Contains(m.Tags, old) {
				t.Errorf("%s lost tag %q: before %v, after %v", rel, old, before[rel], m.Tags)
			}
		}
		subjects := fieldStrings(fields["XMP-dc:Subject"])
		for _, term := range []string{"Anlass", "Hochzeit"} {
			if !slices.Contains(subjects, term) {
				t.Errorf("%s XMP-dc:Subject = %v, want %s", rel, subjects, term)
			}
		}
		if hs := fieldStrings(fields["XMP-lr:HierarchicalSubject"]); !slices.Contains(hs, "Anlass|Hochzeit") {
			t.Errorf("%s XMP-lr:HierarchicalSubject = %v, want Anlass|Hochzeit", rel, hs)
		}
	}
}

func TestRemovingAnlassGeburtstagKeepsAnlassWhileAnlassHochzeitRemainsAndKeepsForeignKeywords(t *testing.T) {
	e := tagEnv(t)
	const rel = "2024/07/tagged.jpg"
	ctx := context.Background()
	// The fixture already carries Anlass/Geburtstag, so assigning it may be a no-op.
	for _, tag := range []string{"Anlass/Hochzeit", "Anlass/Geburtstag"} {
		if res := e.p.WriteFile(ctx, rel, metadata.Change{AddTags: []string{tag}}); res.State == StateFailed {
			t.Fatalf("assign %s: %+v", tag, res)
		}
	}

	if res := e.p.WriteFile(ctx, rel, metadata.Change{RemoveTags: []string{"Anlass/Geburtstag"}}); res.State != StateOK {
		t.Fatalf("remove: %+v", res)
	}

	tags := e.tagsOf(t, rel)
	for _, want := range []string{"Anlass/Hochzeit", "Familie", "Sommer", "Garten"} {
		if !slices.Contains(tags, want) {
			t.Errorf("tags %v lack %s", tags, want)
		}
	}
	if slices.Contains(tags, "Anlass/Geburtstag") {
		t.Errorf("tags %v still hold Anlass/Geburtstag", tags)
	}
	_, fields, err := e.p.readMetaFields(ctx, e.abs(rel))
	if err != nil {
		t.Fatal(err)
	}
	subjects := fieldStrings(fields["XMP-dc:Subject"])
	if !slices.Contains(subjects, "Anlass") || !slices.Contains(subjects, "Hochzeit") || slices.Contains(subjects, "Geburtstag") {
		t.Errorf("XMP-dc:Subject = %v, want Anlass and Hochzeit but not Geburtstag", subjects)
	}
}

func fieldStrings(v any) []string {
	switch x := v.(type) {
	case string:
		return []string{x}
	case []any:
		var out []string
		for _, it := range x {
			if s, ok := it.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

func TestAFileThatAlreadyCarriesTheTagIsSkippedWithMtimeAndBytesUnchanged(t *testing.T) {
	e := tagEnv(t)
	ctx := context.Background()
	const tagged = "2024/07/tagged.jpg"
	if res := e.p.WriteFile(ctx, tagged, metadata.Change{AddTags: []string{"Anlass/Hochzeit"}}); res.State != StateOK {
		t.Fatalf("first assign: %+v", res)
	}
	backupsBefore := listing(t, e.p.BackupDir)

	cases := []struct {
		name, rel string
		c         metadata.Change
	}{
		{"assigning a carried tag", tagged, metadata.Change{AddTags: []string{"Anlass/Hochzeit"}}},
		{"removing a tag the file does not carry", "2024/07/sample.jpg", metadata.Change{RemoveTags: []string{"Anlass/Geburtstag"}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			before := readFile(t, e.abs(tc.rel))
			fi, _ := os.Stat(e.abs(tc.rel))

			res := e.p.WriteFile(ctx, tc.rel, tc.c)

			if res.State != StateSkipped || res.Reason != reasonCurrent {
				t.Errorf("result = %+v, want skipped as already current", res)
			}
			after, _ := os.Stat(e.abs(tc.rel))
			if !after.ModTime().Equal(fi.ModTime()) || !bytes.Equal(readFile(t, e.abs(tc.rel)), before) {
				t.Error("file mtime or bytes changed")
			}
			if got := listing(t, e.p.BackupDir); !slices.Equal(got, backupsBefore) {
				t.Errorf("backup dir changed:\nbefore %v\nafter  %v", backupsBefore, got)
			}
		})
	}
}

func TestATagJobResolvesSingleRangeDayAndMonthSelectionsIncludingFilesNeverListed(t *testing.T) {
	ctx := context.Background()
	e := tagEnv(t)
	if _, err := e.p.Index.Run(ctx, nil); err != nil {
		t.Fatal(err)
	}
	type row struct {
		id             int64
		rel, day, mont string
	}
	rs, err := e.db.Query(`SELECT id, rel_path, COALESCE(substr(taken_at,1,10),''), COALESCE(substr(taken_at,1,7),'') FROM photos
		ORDER BY COALESCE(taken_at, '') DESC, id DESC`)
	if err != nil {
		t.Fatal(err)
	}
	var rows []row
	for rs.Next() {
		var r row
		if err := rs.Scan(&r.id, &r.rel, &r.day, &r.mont); err != nil {
			t.Fatal(err)
		}
		rows = append(rows, r)
	}
	rs.Close()
	if len(rows) < 4 {
		t.Fatalf("only %d indexed fixture files", len(rows))
	}
	pick := func(keep func(row) bool) []string {
		var out []string
		for _, r := range rows {
			if keep(r) {
				out = append(out, r.rel)
			}
		}
		return out
	}
	// rows is in grid order, which is what a range follows.
	first := rows[0]
	cases := []struct {
		name string
		sel  library.Selection
		want []string
	}{
		{"single", library.Selection{IncludeIDs: []int64{first.id}}, []string{first.rel}},
		{"range", library.Selection{Filter: library.SelectionFilter{Ranges: [][2]int64{{rows[1].id, rows[2].id}}}},
			[]string{rows[1].rel, rows[2].rel}},
		{"day", library.Selection{Filter: library.SelectionFilter{Groups: []string{first.day}}},
			pick(func(r row) bool { return r.day == first.day })},
		{"month", library.Selection{Filter: library.SelectionFilter{Groups: []string{first.mont}}, ExcludeIDs: []int64{first.id}},
			pick(func(r row) bool { return r.mont == first.mont && r.id != first.id })},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := library.ResolveSelection(ctx, e.db, tc.sel)
			if err != nil {
				t.Fatal(err)
			}
			var rels []string
			for _, r := range res {
				rels = append(rels, r.RelPath)
			}
			sort.Strings(rels)
			sort.Strings(tc.want)
			if !slices.Equal(rels, tc.want) || len(rels) == 0 {
				t.Fatalf("resolved %v, want %v", rels, tc.want)
			}

			tag := "Auswahl/" + tc.name
			job, items := runTagJob(t, e, metadata.Change{AddTags: []string{tag}}, rels)
			if job.Total != len(rels) || job.Failed != 0 {
				t.Fatalf("job = %+v, items = %+v", job, items)
			}
			var written []string
			for _, it := range items {
				if it.State == "ok" {
					written = append(written, it.RelPath)
				}
			}
			for _, r := range rows {
				// A broken file is part of the selection but cannot carry a tag.
				m, err := e.p.readMeta(ctx, e.abs(r.rel))
				got := err == nil && slices.Contains(m.Tags, tag)
				if want := slices.Contains(written, r.rel); got != want {
					t.Errorf("%s tagged = %v, want %v", r.rel, got, want)
				}
			}
			if len(written) < len(rels)-1 {
				t.Errorf("written %v of selection %v: at most the broken file may be missing", written, rels)
			}
		})
	}
}
