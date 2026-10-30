package library

import (
	"database/sql"
)

// Result counts what one scan did.
type Result struct {
	Seen    int // media files found below the root
	Read    int // files handed to ExifTool
	Skipped int // new or changed files left for a later scan because they are still settling
	Removed int // index rows dropped because their file is gone
	Errors  int // files ExifTool could not read in this scan
}

// UnreadableFile is an indexed file whose last read failed.
type UnreadableFile struct {
	RelPath string
	Error   string
}

// Count returns the number of indexed files.
func Count(db *sql.DB) (int, error) {
	var n int
	err := db.QueryRow(`SELECT count(*) FROM photos`).Scan(&n)
	return n, err
}

// Unreadable returns up to limit files with a read error, ordered by path, and their total.
// A negative limit returns all of them.
func Unreadable(db *sql.DB, limit int) ([]UnreadableFile, int, error) {
	var total int
	if err := db.QueryRow(`SELECT count(*) FROM photos WHERE read_error IS NOT NULL`).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := db.Query(`SELECT rel_path, read_error FROM photos WHERE read_error IS NOT NULL
		ORDER BY rel_path LIMIT ?`, limit)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []UnreadableFile{}
	for rows.Next() {
		var u UnreadableFile
		if err := rows.Scan(&u.RelPath, &u.Error); err != nil {
			return nil, 0, err
		}
		out = append(out, u)
	}
	return out, total, rows.Err()
}
