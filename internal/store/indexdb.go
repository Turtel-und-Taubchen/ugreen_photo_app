package store

import (
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// indexSchemaVersion must change whenever indexSchema does. index.db is a cache that a scan
// rebuilds from the files, so a mismatch is resolved by starting over, never by migrating.
const indexSchemaVersion = 1

var indexSchema = []string{
	`CREATE TABLE photos (
		id              INTEGER PRIMARY KEY,
		rel_path        TEXT NOT NULL UNIQUE,
		folder          TEXT NOT NULL,
		media_type      TEXT NOT NULL,
		format          TEXT NOT NULL,
		size            INTEGER NOT NULL,
		mtime_ns        INTEGER NOT NULL,
		taken_at        TEXT,
		taken_source    TEXT NOT NULL,
		gps_lat         REAL, gps_lon REAL,
		loc_sublocation TEXT, loc_city TEXT, loc_state TEXT, loc_country TEXT,
		loc_label       TEXT,
		review_state    TEXT NOT NULL DEFAULT 'open',
		read_error      TEXT,
		scanned_at      TEXT NOT NULL
	)`,
	`CREATE TABLE photo_tags (
		photo_id INTEGER NOT NULL REFERENCES photos(id) ON DELETE CASCADE,
		tag_path TEXT NOT NULL,
		PRIMARY KEY (photo_id, tag_path)
	)`,
	`CREATE INDEX photos_taken ON photos(taken_at DESC, id DESC)`,
	`CREATE INDEX photos_folder ON photos(folder)`,
	`CREATE INDEX photo_tags_tag ON photo_tags(tag_path)`,
}

// openSQLite opens path with the settings every database here shares. DSN pragmas run on
// each new pooled connection; foreign_keys is per connection in SQLite, so a one-off Exec
// would leave the other connections without ON DELETE CASCADE.
func openSQLite(path string) (*sql.DB, error) {
	return sql.Open("sqlite", "file:"+filepath.ToSlash(path)+
		"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)")
}

// OpenIndexDB opens <dataDir>/index.db and creates its schema on first use. A file written
// by another schema version is deleted (with its -wal/-shm) and recreated empty.
func OpenIndexDB(dataDir string) (*sql.DB, error) {
	path := filepath.Join(dataDir, "index.db")
	db, err := openSQLite(path)
	if err != nil {
		return nil, fmt.Errorf("index database %q cannot be opened: %v", path, err)
	}
	current, empty, err := indexState(db)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("index database %q: %v", path, err)
	}
	if current {
		return db, nil
	}
	if !empty {
		db.Close()
		for _, p := range []string{path, path + "-wal", path + "-shm"} {
			if err := os.Remove(p); err != nil && !errors.Is(err, fs.ErrNotExist) {
				return nil, fmt.Errorf("index database %q cannot be reset: %v", path, err)
			}
		}
		if db, err = openSQLite(path); err != nil {
			return nil, fmt.Errorf("index database %q cannot be opened: %v", path, err)
		}
	}
	if err := createIndexSchema(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("index database %q: %v", path, err)
	}
	return db, nil
}

// indexState reports whether the file carries this build's schema, or else whether it holds
// nothing at all (a new file), which can take the schema without being deleted first.
func indexState(db *sql.DB) (current, empty bool, err error) {
	var version, objects int
	if err := db.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil {
		return false, false, err
	}
	if version == indexSchemaVersion {
		return true, false, nil
	}
	if err := db.QueryRow(`SELECT count(*) FROM sqlite_master`).Scan(&objects); err != nil {
		return false, false, err
	}
	return false, version == 0 && objects == 0, nil
}

func createIndexSchema(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, stmt := range indexSchema {
		if _, err := tx.Exec(stmt); err != nil {
			return err
		}
	}
	// PRAGMA user_version takes no bound parameter; the value is a constant int.
	if _, err := tx.Exec(fmt.Sprintf(`PRAGMA user_version = %d`, indexSchemaVersion)); err != nil {
		return err
	}
	return tx.Commit()
}
