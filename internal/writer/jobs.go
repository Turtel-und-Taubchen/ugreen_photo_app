package writer

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"photocurator/internal/metadata"
)

// Job status values.
const (
	JobQueued  = "queued"
	JobRunning = "running"
	JobDone    = "done"
	JobAborted = "aborted"
)

const (
	itemPending = "pending"
	reasonAbort = "abgebrochen"
)

// Job is one row of the queue; the counters always agree with its items.
type Job struct {
	ID         int64
	Username   string
	Action     string
	Status     string
	Total      int
	Done       int
	OK         int
	Skipped    int
	Failed     int
	CreatedAt  time.Time
	StartedAt  time.Time // zero until the job runs
	FinishedAt time.Time // zero until the job ends
}

// JobItem is the outcome of one file of a job; State is "pending" until it was handled.
type JobItem struct {
	RelPath string
	State   string
	Reason  string
}

// Queue runs write jobs in the order they were created, one at a time and one file at a time.
// Jobs and their files live in DB, so a restart never resumes a job on its own.
type Queue struct {
	DB       *sql.DB // app.db
	Pipeline *Pipeline

	mu      sync.Mutex
	started bool
	wake    chan struct{}
	stopped chan struct{}
	// jobEnded runs after a job reached its final status (tests only).
	jobEnded func(id int64)
}

func unixOrZero(v sql.NullInt64) time.Time {
	if !v.Valid {
		return time.Time{}
	}
	return time.Unix(v.Int64, 0)
}

// Create queues a job for exactly the given files; later changes to the library do not alter
// its list. Duplicate paths count once.
func (q *Queue) Create(ctx context.Context, user, action string, c metadata.Change, rels []string) (int64, error) {
	payload, err := json.Marshal(c)
	if err != nil {
		return 0, err
	}
	seen := make(map[string]bool, len(rels))
	var list []string
	for _, r := range rels {
		if !seen[r] {
			seen[r] = true
			list = append(list, r)
		}
	}
	now := time.Now().Unix()
	tx, err := q.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	res, err := tx.ExecContext(ctx, `INSERT INTO jobs (username, action, payload, status, total, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`, user, action, string(payload), JobQueued, len(list), now)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	for _, r := range list {
		if _, err := tx.ExecContext(ctx, `INSERT INTO job_items (job_id, rel_path, state, updated_at)
			VALUES (?, ?, ?, ?)`, id, r, itemPending, now); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	q.signal()
	return id, nil
}

func (q *Queue) signal() {
	q.mu.Lock()
	w := q.wake
	q.mu.Unlock()
	if w == nil {
		return
	}
	select {
	case w <- struct{}{}:
	default: // a wake-up is already pending
	}
}

// Start restores files an interrupted write left behind, closes every job the previous run left
// open and then starts the runner. Nothing queued before the restart is ever started by it.
func (q *Queue) Start(ctx context.Context) error {
	q.mu.Lock()
	if q.started {
		q.mu.Unlock()
		return errors.New("queue already started")
	}
	q.started = true
	q.mu.Unlock()

	if _, err := q.Pipeline.Recover(ctx); err != nil {
		slog.Error("recover interrupted writes", "err", err)
	}
	if err := q.abortOpen(ctx); err != nil {
		return err
	}
	q.mu.Lock()
	q.wake = make(chan struct{}, 1)
	q.stopped = make(chan struct{})
	q.mu.Unlock()
	go q.run(ctx)
	return nil
}

// Wait blocks until the runner has stopped after Start's ctx ended.
func (q *Queue) Wait() {
	q.mu.Lock()
	s := q.stopped
	q.mu.Unlock()
	if s != nil {
		<-s
	}
}

func (q *Queue) abortOpen(ctx context.Context) error {
	tx, err := q.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	now := time.Now().Unix()
	// Counters first: the open items are still pending here.
	if _, err := tx.ExecContext(ctx, `UPDATE jobs SET
			skipped = skipped + (SELECT COUNT(*) FROM job_items WHERE job_id = jobs.id AND state = ?),
			done = total, status = ?, finished_at = ?
		WHERE status IN (?, ?)`, itemPending, JobAborted, now, JobRunning, JobQueued); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE job_items SET state = 'skipped', reason = ?, updated_at = ?
		WHERE state = ? AND job_id IN (SELECT id FROM jobs WHERE status = ?)`,
		reasonAbort, now, itemPending, JobAborted); err != nil {
		return err
	}
	return tx.Commit()
}

func (q *Queue) run(ctx context.Context) {
	defer close(q.stopped)
	for {
		for ctx.Err() == nil {
			ok, err := q.runNext(ctx)
			if err != nil {
				slog.Error("write job", "err", err)
				break
			}
			if !ok {
				break
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-q.wake:
		}
	}
}

// runNext runs the oldest queued job and reports whether it ran to its end.
func (q *Queue) runNext(ctx context.Context) (bool, error) {
	var id int64
	var payload string
	err := q.DB.QueryRowContext(ctx, `SELECT id, payload FROM jobs WHERE status = ? ORDER BY id LIMIT 1`, JobQueued).
		Scan(&id, &payload)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	var c metadata.Change
	if err := json.Unmarshal([]byte(payload), &c); err != nil {
		return false, fmt.Errorf("job %d payload: %w", id, err)
	}
	if _, err := q.DB.ExecContext(ctx, `UPDATE jobs SET status = ?, started_at = ? WHERE id = ?`,
		JobRunning, time.Now().Unix(), id); err != nil {
		return false, err
	}
	rels, err := q.pending(ctx, id)
	if err != nil {
		return false, err
	}

	for _, rel := range rels {
		// An error in one file is its result, not a reason to stop the job.
		res := q.Pipeline.WriteFile(ctx, rel, c)
		// Stored even when ctx ended meanwhile: the file is already written or restored.
		if err := q.record(context.WithoutCancel(ctx), id, rel, res); err != nil {
			return false, err
		}
		if ctx.Err() != nil {
			return false, nil // stays running; the next Start aborts it
		}
	}
	if _, err := q.DB.ExecContext(ctx, `UPDATE jobs SET status = ?, finished_at = ? WHERE id = ?`,
		JobDone, time.Now().Unix(), id); err != nil {
		return false, err
	}
	if q.jobEnded != nil {
		q.jobEnded(id)
	}
	return true, nil
}

func (q *Queue) pending(ctx context.Context, id int64) ([]string, error) {
	rows, err := q.DB.QueryContext(ctx, `SELECT rel_path FROM job_items WHERE job_id = ? AND state = ? ORDER BY rowid`, id, itemPending)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var rels []string
	for rows.Next() {
		var r string
		if err := rows.Scan(&r); err != nil {
			return nil, err
		}
		rels = append(rels, r)
	}
	return rels, rows.Err()
}

// record stores the item state and the job counters together, so a reader never sees one
// without the other.
func (q *Queue) record(ctx context.Context, id int64, rel string, res Result) error {
	tx, err := q.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `UPDATE job_items SET state = ?, reason = ?, updated_at = ?
		WHERE job_id = ? AND rel_path = ?`, string(res.State), res.Reason, time.Now().Unix(), id, rel); err != nil {
		return err
	}
	col := "failed"
	switch res.State {
	case StateOK:
		col = "ok"
	case StateSkipped:
		col = "skipped"
	}
	if _, err := tx.ExecContext(ctx, `UPDATE jobs SET done = done + 1, `+col+` = `+col+` + 1 WHERE id = ?`, id); err != nil {
		return err
	}
	return tx.Commit()
}

const jobColumns = `id, username, action, status, total, done, ok, skipped, failed, created_at, started_at, finished_at`

type rowScanner interface{ Scan(...any) error }

func scanJob(r rowScanner) (Job, error) {
	var j Job
	var created int64
	var started, finished sql.NullInt64
	if err := r.Scan(&j.ID, &j.Username, &j.Action, &j.Status, &j.Total, &j.Done, &j.OK, &j.Skipped, &j.Failed,
		&created, &started, &finished); err != nil {
		return Job{}, err
	}
	j.CreatedAt = time.Unix(created, 0)
	j.StartedAt, j.FinishedAt = unixOrZero(started), unixOrZero(finished)
	return j, nil
}

// List returns the n newest jobs, newest first.
func (q *Queue) List(ctx context.Context, n int) ([]Job, error) {
	rows, err := q.DB.QueryContext(ctx, `SELECT `+jobColumns+` FROM jobs ORDER BY id DESC LIMIT ?`, n)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Job
	for rows.Next() {
		j, err := scanJob(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, j)
	}
	return out, rows.Err()
}

// Get returns a job with its files in the order they were queued; sql.ErrNoRows if id is unknown.
func (q *Queue) Get(ctx context.Context, id int64) (Job, []JobItem, error) {
	j, err := scanJob(q.DB.QueryRowContext(ctx, `SELECT `+jobColumns+` FROM jobs WHERE id = ?`, id))
	if err != nil {
		return Job{}, nil, err
	}
	rows, err := q.DB.QueryContext(ctx, `SELECT rel_path, state, reason FROM job_items WHERE job_id = ? ORDER BY rowid`, id)
	if err != nil {
		return Job{}, nil, err
	}
	defer rows.Close()
	var items []JobItem
	for rows.Next() {
		var it JobItem
		if err := rows.Scan(&it.RelPath, &it.State, &it.Reason); err != nil {
			return Job{}, nil, err
		}
		items = append(items, it)
	}
	return j, items, rows.Err()
}
