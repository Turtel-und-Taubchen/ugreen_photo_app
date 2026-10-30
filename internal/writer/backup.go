package writer

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const dateLayout = "2006-01-02"

// Backups keeps one dated copy of an original per retention period:
// <Dir>/<YYYY-MM-DD>/<rel_path>, with a row in app.db's backups table.
type Backups struct {
	DB        *sql.DB // app.db
	Root      string  // library root holding the originals
	Dir       string  // backup folder
	Retention int     // days
	Now       func() time.Time
}

// cutoff is the start of the oldest day that still counts as inside the retention period.
// Ensure and Prune both decide with it, so a copy is never called young by one and old by the other.
func (b *Backups) cutoff(now time.Time) time.Time {
	y, m, d := now.AddDate(0, 0, -b.Retention).Date()
	return time.Date(y, m, d, 0, 0, 0, 0, now.Location())
}

// inside returns the cleaned path of rel below base, or an error when it would leave base.
func inside(base, rel string) (string, error) {
	base = filepath.Clean(base)
	p := filepath.Clean(filepath.Join(base, filepath.FromSlash(rel)))
	if !strings.HasPrefix(p, base+string(filepath.Separator)) {
		return "", fmt.Errorf("path %q is outside %s", rel, base)
	}
	return p, nil
}

// Ensure makes sure a backup of the original exists that is younger than the retention period
// and returns its path. created is false when an existing copy was kept.
func (b *Backups) Ensure(relPath string) (path string, created bool, err error) {
	src, err := inside(b.Root, relPath)
	if err != nil {
		return "", false, err
	}
	now := b.Now()
	var stored string
	err = b.DB.QueryRow(`SELECT backup_path FROM backups WHERE rel_path = ? AND created_at >= ?
		ORDER BY created_at DESC LIMIT 1`, relPath, b.cutoff(now).Unix()).Scan(&stored)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return "", false, err
	}
	if err == nil {
		if existing, perr := inside(b.Dir, stored); perr == nil {
			if _, serr := os.Stat(existing); serr == nil {
				return existing, false, nil
			} else if !errors.Is(serr, fs.ErrNotExist) {
				return "", false, serr
			}
		}
	}

	dst, err := inside(b.Dir, now.Format(dateLayout)+"/"+relPath)
	if err != nil {
		return "", false, err
	}
	info, err := os.Stat(src)
	if err != nil {
		return "", false, err
	}
	if err := preCopy(dst, src, info.ModTime()); err != nil {
		return "", false, err
	}
	stored = filepath.ToSlash(strings.TrimPrefix(dst, filepath.Clean(b.Dir)+string(filepath.Separator)))
	if _, err := b.DB.Exec(`INSERT INTO backups (rel_path, backup_path, created_at) VALUES (?, ?, ?)`,
		relPath, stored, now.Unix()); err != nil {
		return "", false, err
	}
	return dst, true, nil
}

// Prune deletes the date folders older than the retention period, in the backup folder and in
// .inflight, together with their rows. Anything else in the backup folder is left alone.
func (b *Backups) Prune(now time.Time) error {
	limit := b.cutoff(now)
	var errs []error
	for _, parent := range []string{b.Dir, filepath.Join(b.Dir, inflightDir)} {
		entries, err := os.ReadDir(parent)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			errs = append(errs, err)
			continue
		}
		for _, e := range entries {
			day, perr := time.ParseInLocation(dateLayout, e.Name(), now.Location())
			if !e.IsDir() || perr != nil || !day.Before(limit) {
				continue
			}
			if err := os.RemoveAll(filepath.Join(parent, e.Name())); err != nil {
				errs = append(errs, err)
			}
		}
	}
	if _, err := b.DB.Exec(`DELETE FROM backups WHERE created_at < ?`, limit.Unix()); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

// Exclusive runs fn while no write job runs and none starts.
func (q *Queue) Exclusive(ctx context.Context, fn func() error) error {
	q.exclusive.Lock()
	defer q.exclusive.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	return fn()
}
