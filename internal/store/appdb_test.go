package store

import "testing"

func TestOpeningTwiceKeepsExistingRowsAndSchemaVersion(t *testing.T) {
	dir := t.TempDir()
	db, err := OpenAppDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO sessions VALUES ('h','u',1,1)`); err != nil {
		t.Fatal(err)
	}
	db.Close()

	db, err = OpenAppDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var n, version int
	if err := db.QueryRow(`SELECT count(*) FROM sessions`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("rows = %d, err = %v", n, err)
	}
	db.QueryRow(`PRAGMA user_version`).Scan(&version)
	if version != len(migrations) {
		t.Fatalf("version = %d", version)
	}
}

func TestTheDatabaseRunsInWriteAheadLogMode(t *testing.T) {
	db, err := OpenAppDB(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var mode string
	db.QueryRow(`PRAGMA journal_mode`).Scan(&mode)
	if mode != "wal" {
		t.Fatalf("journal_mode = %q", mode)
	}
}
