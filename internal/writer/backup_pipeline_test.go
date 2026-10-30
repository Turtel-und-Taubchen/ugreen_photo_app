package writer

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"photocurator/internal/metadata"
)

func TestWriteJobLeavesBackupIdenticalToTheOriginalBeforeTheJob(t *testing.T) {
	e := newEnv(t, true)
	now := time.Date(2024, 9, 10, 15, 30, 0, 0, time.Local)
	e.p.Backups = newBackups(t, e, &now)
	q := newQueue(t, e)
	ended := startQueue(t, q)
	original := readFile(t, e.abs(sampleRel))

	id, err := q.Create(context.Background(), "anna", "tags", metadata.Change{AddTags: []string{"Urlaub"}}, []string{sampleRel})
	if err != nil {
		t.Fatal(err)
	}
	<-ended

	if job, _ := mustGet(t, q, id); job.OK != 1 {
		t.Fatalf("job = %+v, want the file written", job)
	}
	if bytes.Equal(readFile(t, e.abs(sampleRel)), original) {
		t.Fatal("the job did not change the file")
	}
	backup := filepath.Join(e.p.BackupDir, "2024-09-10", filepath.FromSlash(sampleRel))
	if !bytes.Equal(readFile(t, backup), original) {
		t.Error("dated backup differs from the original before the job")
	}
}

func TestFailedBackupLeavesFileUntouchedAndFailsWithReason(t *testing.T) {
	e := newEnv(t, true)
	now := time.Date(2024, 9, 10, 15, 30, 0, 0, time.Local)
	e.p.Backups = newBackups(t, e, &now)
	// A regular file where the backup folder of 2024/07 belongs makes MkdirAll fail, even for
	// root; the files of 2024/08 in the same job can still be backed up.
	blocker := filepath.Join(e.p.BackupDir, "2024-09-10", "2024", "07")
	if err := os.MkdirAll(filepath.Dir(blocker), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	original := readFile(t, e.abs(sampleRel))
	q := newQueue(t, e)
	ended := startQueue(t, q)

	rels := []string{sampleRel, goodRels[2], goodRels[3]}
	id, err := q.Create(context.Background(), "anna", "tags", metadata.Change{AddTags: []string{"Urlaub"}}, rels)
	if err != nil {
		t.Fatal(err)
	}
	<-ended

	_, items := mustGet(t, q, id)
	for _, it := range items {
		if it.RelPath == sampleRel {
			if it.State != string(StateFailed) || !strings.HasPrefix(it.Reason, "Backup fehlgeschlagen") {
				t.Errorf("%s: state %q reason %q, want failed with backup reason", it.RelPath, it.State, it.Reason)
			}
		} else if it.State != string(StateOK) {
			t.Errorf("%s: state %q reason %q, want ok", it.RelPath, it.State, it.Reason)
		}
	}
	if len(items) != len(rels) {
		t.Errorf("%d items, want %d", len(items), len(rels))
	}
	if !bytes.Equal(readFile(t, e.abs(sampleRel)), original) {
		t.Error("file bytes changed although its backup failed")
	}
	e.assertNoInflightCopies(t)
}

func TestReadBackFailureRestoresStateBeforeThisWrite(t *testing.T) {
	e := newEnv(t, true)
	now := time.Date(2024, 9, 10, 15, 30, 0, 0, time.Local)
	e.p.Backups = newBackups(t, e, &now)
	ctx := context.Background()
	original := readFile(t, e.abs(sampleRel))

	if res := e.p.WriteFile(ctx, sampleRel, metadata.Change{AddTags: []string{"A"}}); res.State != StateOK {
		t.Fatalf("first write = %+v, want ok", res)
	}
	withA := readFile(t, e.abs(sampleRel))
	backup := filepath.Join(e.p.BackupDir, "2024-09-10", filepath.FromSlash(sampleRel))
	backupBefore := readFile(t, backup)

	// A day later the dated copy is still inside the retention period and holds the state
	// before tag A; restoring from it would silently undo the first write.
	now = now.AddDate(0, 0, 1)
	e.p.AfterWrite = func(abs string) error {
		return e.p.Exif.Write(context.Background(), abs, []string{"-XMP-dc:Subject+=Fremd"})
	}
	res := e.p.WriteFile(ctx, sampleRel, metadata.Change{AddTags: []string{"B"}})
	if res.State != StateFailed || !strings.Contains(res.Reason, "Read-back") {
		t.Errorf("second write = %+v, want failed with read-back reason", res)
	}

	got := readFile(t, e.abs(sampleRel))
	if bytes.Equal(got, original) {
		t.Error("file was rolled back to the dated backup, losing tag A")
	} else if !bytes.Equal(got, withA) {
		t.Error("file differs from its state right before the failed write")
	}
	if !bytes.Equal(readFile(t, backup), backupBefore) {
		t.Error("dated backup changed")
	}
	if _, err := os.Stat(filepath.Join(e.p.BackupDir, "2024-09-11")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("a second dated backup was taken: %v", err)
	}
	e.assertNoInflightCopies(t)
}
