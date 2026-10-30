package store

import "testing"

func insertPhoto(t *testing.T, dir string) {
	t.Helper()
	db, err := OpenIndexDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`INSERT INTO photos (rel_path, folder, media_type, format, size, mtime_ns,
		taken_source, scanned_at) VALUES ('a.jpg', '', 'image', 'jpeg', 1, 1, 'mtime', 'now')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO photo_tags VALUES (1, 'Familie')`); err != nil {
		t.Fatal(err)
	}
}

func countRows(t *testing.T, dir string) (photos, version, foreignKeys int) {
	t.Helper()
	db, err := OpenIndexDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := db.QueryRow(`SELECT count(*) FROM photos`).Scan(&photos); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`PRAGMA foreign_keys`).Scan(&foreignKeys); err != nil {
		t.Fatal(err)
	}
	return photos, version, foreignKeys
}

func TestAnIndexWithAForeignSchemaVersionIsRebuiltEmpty(t *testing.T) {
	dir := t.TempDir()
	insertPhoto(t, dir)
	db, err := OpenIndexDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE leftover (x)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`PRAGMA user_version = 99`); err != nil {
		t.Fatal(err)
	}
	db.Close()

	photos, version, fk := countRows(t, dir)
	if photos != 0 || version != indexSchemaVersion || fk != 1 {
		t.Fatalf("photos = %d, version = %d, foreign_keys = %d; want 0, %d, 1", photos, version, fk, indexSchemaVersion)
	}
	db, err = OpenIndexDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var leftovers int
	if err := db.QueryRow(`SELECT count(*) FROM sqlite_master WHERE name = 'leftover'`).Scan(&leftovers); err != nil || leftovers != 0 {
		t.Fatalf("foreign table survived the rebuild (count %d, err %v)", leftovers, err)
	}
}

func TestAnIndexWithTheCurrentSchemaVersionKeepsItsRows(t *testing.T) {
	dir := t.TempDir()
	insertPhoto(t, dir)
	if photos, version, _ := countRows(t, dir); photos != 1 || version != indexSchemaVersion {
		t.Fatalf("photos = %d, version = %d; want 1, %d", photos, version, indexSchemaVersion)
	}
}
