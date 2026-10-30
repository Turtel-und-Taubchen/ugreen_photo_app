package tags

import (
	"context"
	"database/sql"
)

// RecentLimit is the number of recently used tags that are kept and listed.
const RecentLimit = 8

// TouchRecent marks paths as used, in payload order, so each gets a strictly larger seq than the
// one before it, and drops everything but the RecentLimit newest. Paths must already be canonical.
func TouchRecent(ctx context.Context, db *sql.DB, paths []string) error {
	if len(paths) == 0 {
		return nil
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, p := range paths {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO recent_tags (path, seq) VALUES (?, (SELECT COALESCE(MAX(seq), 0) FROM recent_tags) + 1)
			 ON CONFLICT(path) DO UPDATE SET seq = excluded.seq`, p); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx,
		`DELETE FROM recent_tags WHERE path NOT IN (SELECT path FROM recent_tags ORDER BY seq DESC LIMIT ?)`,
		RecentLimit); err != nil {
		return err
	}
	return tx.Commit()
}

// ListRecent returns the most recently used paths, newest first.
func ListRecent(ctx context.Context, db *sql.DB) ([]string, error) {
	rows, err := db.QueryContext(ctx, `SELECT path FROM recent_tags ORDER BY seq DESC LIMIT ?`, RecentLimit)
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
