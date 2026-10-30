package library

import (
	"context"
	"database/sql"
	"errors"
)

// Photo is the part of an index row the list and thumbnail endpoints need.
type Photo struct {
	ID        int64
	RelPath   string
	MediaType string
	Size      int64
	MtimeNs   int64
	TakenAt   *string
}

// ListCursor is the keyset position of the newest-first listing: the last row of a page.
type ListCursor struct {
	TakenAt string // "" for files without a taken date, which sort last
	ID      int64
}

const photoColumns = `id, rel_path, media_type, size, mtime_ns, taken_at`

func scanPhoto(sc interface{ Scan(...any) error }) (Photo, error) {
	var p Photo
	err := sc.Scan(&p.ID, &p.RelPath, &p.MediaType, &p.Size, &p.MtimeNs, &p.TakenAt)
	return p, err
}

// PhotoByID returns the indexed file with that id; ok is false when there is none.
func PhotoByID(db *sql.DB, id int64) (p Photo, ok bool, err error) {
	p, err = scanPhoto(db.QueryRow(`SELECT `+photoColumns+` FROM photos WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return Photo{}, false, nil
	}
	return p, err == nil, err
}

// ListPhotos returns up to limit files ordered by taken date and id, newest first, starting
// after cursor (nil = from the top). Files without a taken date come last.
func ListPhotos(db *sql.DB, after *ListCursor, limit int) ([]Photo, error) {
	q := `SELECT ` + photoColumns + ` FROM photos`
	var args []any
	if after != nil {
		q += ` WHERE (COALESCE(taken_at, ''), id) < (?, ?)`
		args = append(args, after.TakenAt, after.ID)
	}
	q += ` ORDER BY COALESCE(taken_at, '') DESC, id DESC LIMIT ?`
	args = append(args, limit)
	return queryPhotos(context.Background(), db, q, args)
}

// AllPhotos returns every indexed file in the order of ListPhotos.
func AllPhotos(db *sql.DB) ([]Photo, error) {
	return queryPhotos(context.Background(), db, `SELECT `+photoColumns+` FROM photos ORDER BY COALESCE(taken_at, '') DESC, id DESC`, nil)
}

func queryPhotos(ctx context.Context, db *sql.DB, q string, args []any) ([]Photo, error) {
	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Photo{}
	for rows.Next() {
		p, err := scanPhoto(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
