package library

import (
	"context"
	"database/sql"
)

// TagPairs returns a (photoID, path) sequence over photo_tags ordered by photo id, shaped for
// tags.BuildTree, plus an accessor for the first read error. Call the accessor after the
// sequence was consumed; a failed query yields an empty sequence and a non-nil error.
func TagPairs(ctx context.Context, db *sql.DB) (func(yield func(int64, string) bool), func() error) {
	var firstErr error
	seq := func(yield func(int64, string) bool) {
		rows, err := db.QueryContext(ctx, `SELECT photo_id, tag_path FROM photo_tags ORDER BY photo_id`)
		if err != nil {
			firstErr = err
			return
		}
		defer rows.Close()
		for rows.Next() {
			var id int64
			var p string
			if err := rows.Scan(&id, &p); err != nil {
				firstErr = err
				return
			}
			if !yield(id, p) {
				return
			}
		}
		firstErr = rows.Err()
	}
	return seq, func() error { return firstErr }
}

// TagCount is one tag path and the number of selected photos carrying exactly that path.
type TagCount struct {
	Path  string `json:"path"`
	Count int    `json:"count"`
}

// SelectionTagCounts lists the tags of the selected photos with their photo counts, sorted by
// path. The selection stays in SQL; the result is never nil.
func SelectionTagCounts(ctx context.Context, db *sql.DB, s Selection) ([]TagCount, error) {
	out := []TagCount{}
	where, args, ok, err := selectionWhere(ctx, db, s)
	if err != nil || !ok {
		return out, err
	}
	rows, err := db.QueryContext(ctx,
		`SELECT tag_path, COUNT(DISTINCT photo_id) FROM photo_tags
		 WHERE photo_id IN (SELECT id FROM photos WHERE `+where+`)
		 GROUP BY tag_path ORDER BY tag_path`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var c TagCount
		if err := rows.Scan(&c.Path, &c.Count); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// TagSubtreeCount counts distinct photos tagged with path or any tag below it. Prefix matching
// is literal (no LIKE): "_" and "%" are valid tag characters.
func TagSubtreeCount(ctx context.Context, db *sql.DB, path string) (int, error) {
	var n int
	err := db.QueryRowContext(ctx,
		`SELECT COUNT(DISTINCT photo_id) FROM photo_tags
		 WHERE tag_path = ? OR substr(tag_path, 1, length(?) + 1) = ? || '/'`,
		path, path, path).Scan(&n)
	return n, err
}
