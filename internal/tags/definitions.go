package tags

import (
	"context"
	"database/sql"
	"time"
)

// Create stores path and every missing ancestor in one transaction. path must already be
// canonical. created is false when path itself was defined before.
func Create(ctx context.Context, db *sql.DB, path string) (created bool, err error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	now := time.Now().Unix()
	for i := 0; i <= len(path); i++ {
		if i != len(path) && path[i] != '/' {
			continue
		}
		res, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO tag_definitions (path, created_at) VALUES (?, ?)`, path[:i], now)
		if err != nil {
			return false, err
		}
		if i == len(path) {
			n, err := res.RowsAffected()
			if err != nil {
				return false, err
			}
			created = n > 0
		}
	}
	return created, tx.Commit()
}

// List returns all defined paths in byte order.
func List(ctx context.Context, db *sql.DB) ([]string, error) {
	rows, err := db.QueryContext(ctx, `SELECT path FROM tag_definitions ORDER BY path`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// DeleteSubtree removes path and every definition below it and returns the row count.
// Prefix matching is literal (no LIKE): "_" and "%" are valid tag characters.
func DeleteSubtree(ctx context.Context, db *sql.DB, path string) (int64, error) {
	res, err := db.ExecContext(ctx,
		`DELETE FROM tag_definitions WHERE path = ? OR substr(path, 1, length(?) + 1) = ? || '/'`,
		path, path, path)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
