package writer

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"time"

	"photocurator/internal/metadata"
)

// ActionUndo is the action of a job that reverts another job.
const ActionUndo = "undo"

var (
	// ErrAlreadyUndone: an undo job exists for the source job; it is never undone twice.
	ErrAlreadyUndone = errors.New("job was already undone")
	// ErrJobActive: the source job is still queued or running.
	ErrJobActive = errors.New("job is still queued or running")
)

const reasonConflict = "Konflikt: seit dem Job geändert"

// Undo queues a job that reverts the files the source job wrote successfully. The source job is
// unknown: sql.ErrNoRows.
func (q *Queue) Undo(ctx context.Context, sourceID int64, user string) (int64, error) {
	// Two SQLite read transactions that both want to write fail with SQLITE_BUSY instead of
	// waiting; the unique index stays as the guard against any other writer.
	q.undoMu.Lock()
	defer q.undoMu.Unlock()
	tx, err := q.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	var status string
	if err := tx.QueryRowContext(ctx, `SELECT status FROM jobs WHERE id = ?`, sourceID).Scan(&status); err != nil {
		return 0, err
	}
	if status != JobDone && status != JobAborted {
		return 0, ErrJobActive
	}
	var undone int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM jobs WHERE undo_of = ?`, sourceID).Scan(&undone); err != nil {
		return 0, err
	}
	if undone > 0 {
		return 0, ErrAlreadyUndone
	}
	var total int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM job_items WHERE job_id = ? AND state = ?`,
		sourceID, string(StateOK)).Scan(&total); err != nil {
		return 0, err
	}

	now := time.Now().Unix()
	res, err := tx.ExecContext(ctx, `INSERT INTO jobs (username, action, payload, status, total, created_at, undo_of)
		VALUES (?, ?, ?, ?, ?, ?, ?)`, user, ActionUndo, fmt.Sprintf(`{"undo_of":%d}`, sourceID), JobQueued, total, now, sourceID)
	if err != nil {
		// A concurrent undo that committed first violates the unique index.
		var n int
		if q.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM jobs WHERE undo_of = ?`, sourceID).Scan(&n) == nil && n > 0 {
			return 0, ErrAlreadyUndone
		}
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO job_items (job_id, rel_path, state, updated_at)
		SELECT ?, rel_path, ?, ? FROM job_items WHERE job_id = ? AND state = ? ORDER BY rowid`,
		id, itemPending, now, sourceID, string(StateOK)); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		if q.DB.QueryRowContext(ctx, `SELECT 1 FROM jobs WHERE undo_of = ? AND id <> ?`, sourceID, id).Scan(new(int)) == nil {
			return 0, ErrAlreadyUndone
		}
		return 0, err
	}
	q.signal()
	return id, nil
}

// sourceItems returns the items of the job an undo job reverts, by path.
func (q *Queue) sourceItems(ctx context.Context, id int64) (map[string]JobItem, error) {
	_, items, err := q.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	m := make(map[string]JobItem, len(items))
	for _, it := range items {
		m[it.RelPath] = it
	}
	return m, nil
}

// undoFile restores the values src journalled before it. The file is read directly, never the
// index: a file that no longer holds what src wrote was changed since and stays untouched. This
// check takes no backup and no pre-copy; the write itself goes through the normal pipeline.
func (p *Pipeline) undoFile(ctx context.Context, rel string, src JobItem) Result {
	c, ok := undoChange(src)
	if !ok {
		return Result{State: StateSkipped, Reason: "Ursprungsort ohne Koordinaten nicht wiederherstellbar"}
	}
	// A file WriteFile refuses (outside the library, missing, ...) gets its usual result.
	if abs, _, _, reason := p.resolve(rel); reason == "" {
		cur, err := p.readMeta(ctx, abs)
		if err != nil {
			return Result{State: StateFailed, Reason: "Metadaten nicht lesbar: " + err.Error()}
		}
		if !reflect.DeepEqual(JournalOf(cur, AffectedBy(c)), src.New) {
			return Result{State: StateSkipped, Reason: reasonConflict}
		}
	}
	return p.WriteFile(ctx, rel, c)
}

// undoChange is the change that turns what src wrote back into what it replaced. Tags are a set
// difference, so keywords other tools added in between stay.
func undoChange(src JobItem) (metadata.Change, bool) {
	var c metadata.Change
	if src.Old.Tags.Affected {
		old, cur := tagSet(src.Old.Tags.Value), tagSet(src.New.Tags.Value)
		for _, t := range old {
			if !slices.Contains(cur, t) {
				c.AddTags = append(c.AddTags, t)
			}
		}
		for _, t := range cur {
			if !slices.Contains(old, t) {
				c.RemoveTags = append(c.RemoveTags, t)
			}
		}
	}
	if src.Old.Location.Affected {
		c.ClearLocation = true
		if l := src.Old.Location.Value; l != nil {
			if l.Lat == nil || l.Lon == nil {
				return c, false
			}
			c.Location = &metadata.Location{Lat: *l.Lat, Lon: *l.Lon, Label: l.Label,
				Sublocation: l.Sublocation, City: l.City, State: l.State, Country: l.Country}
		}
	}
	if src.Old.ReviewState.Affected {
		// Absent and "open" read the same, so deleting restores a file that had no state.
		if v := src.Old.ReviewState.Value; v != nil && *v != metadata.ReviewOpen {
			c.ReviewState = *v
		} else {
			c.ClearReviewState = true
		}
	}
	return c, true
}

func tagSet(v *[]string) []string {
	if v == nil {
		return nil
	}
	return *v
}
