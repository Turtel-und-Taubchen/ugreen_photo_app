package writer

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"photocurator/internal/metadata"
	"photocurator/internal/store"
)

var goodRels = []string{"2024/07/sample.jpg", "2024/07/tagged.jpg", "2024/08/20240815_143005.jpg", "2024/08/IMG-20240812-WA0001.jpg"}

func newQueue(t *testing.T, e *env) *Queue {
	t.Helper()
	db, err := store.OpenAppDB(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return &Queue{DB: db, Pipeline: e.p}
}

// startQueue runs q until the test ends and delivers the id of every finished job.
func startQueue(t *testing.T, q *Queue) <-chan int64 {
	t.Helper()
	ended := make(chan int64, 8)
	q.jobEnded = func(id int64) { ended <- id }
	ctx, cancel := context.WithCancel(context.Background())
	if err := q.Start(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cancel(); q.Wait() })
	return ended
}

func mustGet(t *testing.T, q *Queue, id int64) (Job, []JobItem) {
	t.Helper()
	j, items, err := q.Get(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	return j, items
}

func TestBrokenFileFailsAloneAndTheRestIsWritten(t *testing.T) {
	e := newEnv(t, true)
	q := newQueue(t, e)
	ended := startQueue(t, q)

	rels := []string{"2024/08/defekt.jpg", "2024/08/fehlt.jpg", goodRels[0], goodRels[1]}
	id, err := q.Create(context.Background(), "anna", "tags", metadata.Change{AddTags: []string{"Urlaub"}}, rels)
	if err != nil {
		t.Fatal(err)
	}
	<-ended

	job, items := mustGet(t, q, id)
	if job.Status != JobDone || job.OK != 2 || job.Failed != 2 || job.Done != 4 || job.Total != 4 {
		t.Errorf("job = %+v, want done with 2 ok, 2 failed", job)
	}
	for i, it := range items {
		wantFailed := i < 2
		if (it.State == "failed") != wantFailed || (it.State == "ok") == wantFailed {
			t.Errorf("item %s state %q", it.RelPath, it.State)
		}
		if wantFailed && it.Reason == "" {
			t.Errorf("failed item %s has no reason", it.RelPath)
		}
	}
	if items[1].Reason != reasonNotFound {
		t.Errorf("missing file reason = %q", items[1].Reason)
	}
}

func TestOnlyOneWriteJobRunsAtATimeAndOthersWait(t *testing.T) {
	e := newEnv(t, true)
	q := newQueue(t, e)
	entered, release := make(chan struct{}), make(chan struct{})
	e.p.AfterWrite = func(string) error {
		entered <- struct{}{}
		<-release
		return nil
	}
	ended := startQueue(t, q)
	ctx := context.Background()
	change := metadata.Change{AddTags: []string{"Urlaub"}}

	first, err := q.Create(ctx, "anna", "tags", change, goodRels[:2])
	if err != nil {
		t.Fatal(err)
	}
	for done := 0; done < 2; done++ {
		<-entered
		if j, _ := mustGet(t, q, first); j.Done != done || j.Status != JobRunning {
			t.Errorf("before file %d: job = %+v, want running with done=%d", done+1, j, done)
		}
		if done == 0 {
			second, err := q.Create(ctx, "anna", "tags", change, goodRels[2:3])
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				if j, _ := mustGet(t, q, second); j.Status != JobDone {
					t.Errorf("second job = %+v, want done", j)
				}
			}()
		}
		jobs, err := q.List(ctx, 10)
		if err != nil {
			t.Fatal(err)
		}
		running := 0
		for _, j := range jobs {
			if j.Status == JobRunning {
				running++
			}
			if j.ID != first && j.Status != JobQueued {
				t.Errorf("job %d is %s while job %d still writes", j.ID, j.Status, first)
			}
		}
		if running != 1 {
			t.Errorf("%d jobs running, want 1", running)
		}
		release <- struct{}{}
	}
	if id := <-ended; id != first {
		t.Fatalf("job %d ended first, want %d", id, first)
	}
	<-entered
	release <- struct{}{}
	<-ended
}

func TestRestartMarksJobsAbortedAndRestoresTheFileInFlight(t *testing.T) {
	e := newEnv(t, true)
	q := newQueue(t, e)
	original := readFile(t, e.abs(goodRels[1]))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	e.p.AfterWrite = func(abs string) error {
		if filepath.Base(abs) != filepath.Base(goodRels[1]) {
			return nil
		}
		cancel() // the process dies with the second file half done
		return errors.New("simulated crash")
	}
	if err := q.Start(ctx); err != nil {
		t.Fatal(err)
	}
	tag := "Urlaub"
	id, err := q.Create(context.Background(), "anna", "tags", metadata.Change{AddTags: []string{tag}}, goodRels)
	if err != nil {
		t.Fatal(err)
	}
	q.Wait()

	e.p.AfterWrite = nil
	restarted := &Queue{DB: q.DB, Pipeline: e.p}
	ctx2, cancel2 := context.WithCancel(context.Background())
	if err := restarted.Start(ctx2); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cancel2(); restarted.Wait() })

	job, items := mustGet(t, restarted, id)
	if job.Status != JobAborted || job.Done != job.Total || job.OK != 1 || job.Failed != 1 || job.Skipped != 2 {
		t.Errorf("job = %+v, want aborted with 1 ok, 1 failed, 2 skipped", job)
	}
	states := make([]string, len(items))
	for i, it := range items {
		states[i] = it.State
	}
	if want := []string{"ok", "failed", "skipped", "skipped"}; !slices.Equal(states, want) {
		t.Errorf("item states %v, want %v", states, want)
	}
	for _, it := range items[2:] {
		if !strings.Contains(it.Reason, "abgebrochen") {
			t.Errorf("item %s reason %q", it.RelPath, it.Reason)
		}
	}
	if !bytes.Equal(readFile(t, e.abs(goodRels[1])), original) {
		t.Error("file in flight does not hold its original bytes")
	}
	meta, err := e.p.readMeta(context.Background(), e.abs(goodRels[0]))
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(meta.Tags, tag) {
		t.Errorf("first file tags %v lack %q", meta.Tags, tag)
	}
	e.assertNoInflightCopies(t)
}

func TestReadOnlyFileInABatchFailsAloneAndTheRestIsWritten(t *testing.T) {
	e := newEnv(t, true)
	q := newQueue(t, e)
	ended := startQueue(t, q)
	readOnly := goodRels[1]
	if err := os.Chmod(e.abs(readOnly), 0o444); err != nil {
		t.Fatal(err)
	}

	id, err := q.Create(context.Background(), "anna", "tags", metadata.Change{AddTags: []string{"Urlaub"}}, goodRels)
	if err != nil {
		t.Fatal(err)
	}
	<-ended

	job, items := mustGet(t, q, id)
	if job.Status != JobDone || job.OK != 3 || job.Failed != 1 || job.Total != 4 {
		t.Errorf("job = %+v, want done with 3 ok, 1 failed", job)
	}
	for _, it := range items {
		if wantFailed := it.RelPath == readOnly; (it.State == "failed") != wantFailed {
			t.Errorf("item %s state %q", it.RelPath, it.State)
		} else if wantFailed && it.Reason == "" {
			t.Errorf("failed item %s has no reason", it.RelPath)
		}
	}
	e.assertNoInflightCopies(t)
}
