package library

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// DayCount, MonthGroup, FolderCount and Groups describe the library for the grid's section index.
type DayCount struct {
	Day   string // "2024-07-12"
	Count int
}

type MonthGroup struct {
	Month string // "2024-07"
	Count int
	Days  []DayCount
}

type FolderCount struct {
	Folder string // "2024/07"
	Count  int
}

type Groups struct {
	Total   int // every indexed file, including those without a taken date
	Months  []MonthGroup
	Folders []FolderCount
}

// Place is the human-readable location written in a file.
type Place struct {
	Label, Sublocation, City, State, Country string
}

// Detail is everything the detail view shows about one file. PrevID is the neighbour shown
// before it in grid order (newer), NextID the one after it (older); nil at the ends.
type Detail struct {
	Photo
	Folder      string
	Format      string
	TakenSource string
	Tags        []string
	Place       Place
	ReviewState string
	ReadError   *string
	PrevID      *int64
	NextID      *int64
}

// GroupCounts counts files per month and day of their taken date, newest first, and per
// folder that follows the JJJJ/MM layout. Files without a taken date only add to Total.
func GroupCounts(ctx context.Context, db *sql.DB) (Groups, error) {
	g := Groups{Months: []MonthGroup{}, Folders: []FolderCount{}}
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM photos`).Scan(&g.Total); err != nil {
		return g, err
	}
	rows, err := db.QueryContext(ctx, `SELECT substr(taken_at, 1, 10) AS day, count(*) FROM photos
		WHERE taken_at IS NOT NULL GROUP BY day ORDER BY day DESC`)
	if err != nil {
		return g, err
	}
	defer rows.Close()
	for rows.Next() {
		var d DayCount
		if err := rows.Scan(&d.Day, &d.Count); err != nil {
			return g, err
		}
		if len(d.Day) < 7 {
			continue
		}
		if n := len(g.Months); n == 0 || g.Months[n-1].Month != d.Day[:7] {
			g.Months = append(g.Months, MonthGroup{Month: d.Day[:7], Days: []DayCount{}})
		}
		m := &g.Months[len(g.Months)-1]
		m.Count += d.Count
		m.Days = append(m.Days, d)
	}
	if err := rows.Err(); err != nil {
		return g, err
	}
	frows, err := db.QueryContext(ctx, `SELECT folder, count(*) FROM photos
		WHERE folder GLOB '[0-9][0-9][0-9][0-9]/[0-9][0-9]' GROUP BY folder ORDER BY folder DESC`)
	if err != nil {
		return g, err
	}
	defer frows.Close()
	for frows.Next() {
		var f FolderCount
		if err := frows.Scan(&f.Folder, &f.Count); err != nil {
			return g, err
		}
		g.Folders = append(g.Folders, f)
	}
	return g, frows.Err()
}

// ListMonth is ListPhotos restricted to files taken in month ("2024-07"). The taken date is
// compared as a range so the photos_taken index serves it.
func ListMonth(ctx context.Context, db *sql.DB, month string, after *ListCursor, limit int) ([]Photo, error) {
	start, err := time.Parse("2006-01", month)
	if err != nil {
		return nil, fmt.Errorf("month %q: %w", month, err)
	}
	q := `SELECT ` + photoColumns + ` FROM photos WHERE taken_at >= ? AND taken_at < ?`
	args := []any{month, start.AddDate(0, 1, 0).Format("2006-01")}
	if after != nil {
		q += ` AND (taken_at, id) < (?, ?)`
		args = append(args, after.TakenAt, after.ID)
	}
	q += ` ORDER BY taken_at DESC, id DESC LIMIT ?`
	args = append(args, limit)
	return queryPhotos(ctx, db, q, args)
}

// PhotoDetail returns the full record with tags and grid neighbours; ok is false for an unknown id.
func PhotoDetail(ctx context.Context, db *sql.DB, id int64) (d Detail, ok bool, err error) {
	var sub, city, state, country, label sql.NullString
	err = db.QueryRowContext(ctx, `SELECT `+photoColumns+`, folder, format, taken_source,
		loc_label, loc_sublocation, loc_city, loc_state, loc_country, review_state, read_error
		FROM photos WHERE id = ?`, id).Scan(&d.ID, &d.RelPath, &d.MediaType, &d.Size, &d.MtimeNs, &d.TakenAt,
		&d.Folder, &d.Format, &d.TakenSource, &label, &sub, &city, &state, &country, &d.ReviewState, &d.ReadError)
	if errors.Is(err, sql.ErrNoRows) {
		return Detail{}, false, nil
	}
	if err != nil {
		return Detail{}, false, err
	}
	d.Place = Place{label.String, sub.String, city.String, state.String, country.String}

	d.Tags = []string{}
	rows, err := db.QueryContext(ctx, `SELECT tag_path FROM photo_tags WHERE photo_id = ? ORDER BY tag_path`, id)
	if err != nil {
		return Detail{}, false, err
	}
	defer rows.Close()
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			return Detail{}, false, err
		}
		d.Tags = append(d.Tags, t)
	}
	if err := rows.Err(); err != nil {
		return Detail{}, false, err
	}

	at := ""
	if d.TakenAt != nil {
		at = *d.TakenAt
	}
	// Same ordering key as ListPhotos, so the neighbours match what the grid shows.
	if d.PrevID, err = neighbour(ctx, db, `> (?, ?) ORDER BY COALESCE(taken_at, '') ASC, id ASC`, at, id); err != nil {
		return Detail{}, false, err
	}
	if d.NextID, err = neighbour(ctx, db, `< (?, ?) ORDER BY COALESCE(taken_at, '') DESC, id DESC`, at, id); err != nil {
		return Detail{}, false, err
	}
	return d, true, nil
}

func neighbour(ctx context.Context, db *sql.DB, cond, at string, id int64) (*int64, error) {
	var n int64
	err := db.QueryRowContext(ctx, `SELECT id FROM photos WHERE (COALESCE(taken_at, ''), id) `+cond+` LIMIT 1`, at, id).Scan(&n)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &n, nil
}
