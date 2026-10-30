// Package store opens the SQLite databases: the durable app.db and the index.db cache.
package store

import (
	"database/sql"
	"fmt"
	"path/filepath"

	_ "modernc.org/sqlite" // pure Go driver: the image has no C toolchain
)

// migrations[i] upgrades schema version i to i+1. Append only; never edit a shipped entry.
var migrations = []string{
	`CREATE TABLE sessions (
		id_hash TEXT PRIMARY KEY,
		username TEXT NOT NULL,
		expires_at INTEGER NOT NULL,
		renewed_at INTEGER NOT NULL
	)`,
	`CREATE TABLE tag_definitions (
		path TEXT PRIMARY KEY,
		created_at INTEGER NOT NULL
	)`,
	`CREATE TABLE geocode_cache (
		key TEXT PRIMARY KEY,
		response TEXT NOT NULL,
		fetched_at INTEGER NOT NULL
	)`,
	`CREATE TABLE jobs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT NOT NULL,
		action TEXT NOT NULL,
		payload TEXT NOT NULL,
		status TEXT NOT NULL,
		total INTEGER NOT NULL,
		done INTEGER NOT NULL DEFAULT 0,
		ok INTEGER NOT NULL DEFAULT 0,
		skipped INTEGER NOT NULL DEFAULT 0,
		failed INTEGER NOT NULL DEFAULT 0,
		created_at INTEGER NOT NULL,
		started_at INTEGER,
		finished_at INTEGER
	);
	CREATE TABLE job_items (
		job_id INTEGER NOT NULL REFERENCES jobs(id),
		rel_path TEXT NOT NULL,
		state TEXT NOT NULL,
		reason TEXT NOT NULL DEFAULT '',
		updated_at INTEGER NOT NULL
	);
	CREATE INDEX job_items_job ON job_items(job_id)`,
}

// OpenAppDB opens <dataDir>/app.db in WAL mode and applies pending migrations.
func OpenAppDB(dataDir string) (*sql.DB, error) {
	path := filepath.Join(dataDir, "app.db")
	db, err := openSQLite(path)
	if err != nil {
		return nil, fmt.Errorf("app database %q cannot be opened: %v", path, err)
	}
	if err := migrate(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("app database %q: %v", path, err)
	}
	return db, nil
}

func migrate(db *sql.DB) error {
	var version int
	if err := db.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil {
		return err
	}
	if version > len(migrations) {
		return fmt.Errorf("schema version %d is newer than this build supports (%d)", version, len(migrations))
	}
	for v := version; v < len(migrations); v++ {
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		// PRAGMA user_version takes no bound parameter; v is an int, so Sprintf is safe.
		if _, err := tx.Exec(migrations[v]); err != nil {
			tx.Rollback()
			return fmt.Errorf("migration %d: %v", v+1, err)
		}
		if _, err := tx.Exec(fmt.Sprintf(`PRAGMA user_version = %d`, v+1)); err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}
