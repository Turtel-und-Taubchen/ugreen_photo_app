package library

import (
	"database/sql"
	"fmt"
	"reflect"
	"testing"

	"photocurator/internal/store"
)

func openPhotosDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := store.OpenIndexDB(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func insertPhoto(t *testing.T, db *sql.DB, id int64, takenAt *string) {
	t.Helper()
	_, err := db.Exec(`INSERT INTO photos (id, rel_path, folder, media_type, format, size, mtime_ns, taken_at, taken_source, scanned_at)
		VALUES (?, ?, '', 'image', 'jpg', ?, ?, ?, 'exif', '2024-01-01T00:00:00Z')`,
		id, fmt.Sprintf("p%d.jpg", id), id*10, id*100, takenAt)
	if err != nil {
		t.Fatal(err)
	}
}

func ptr(s string) *string { return &s }

// seedPhotos: ids 1..7; ids 2 and 4 share a taken date, 3 and 6 have none.
func seedPhotos(t *testing.T) *sql.DB {
	db := openPhotosDB(t)
	insertPhoto(t, db, 1, ptr("2024-01-01T10:00:00"))
	insertPhoto(t, db, 2, ptr("2024-03-01T10:00:00"))
	insertPhoto(t, db, 3, nil)
	insertPhoto(t, db, 4, ptr("2024-03-01T10:00:00"))
	insertPhoto(t, db, 5, ptr("2024-05-01T10:00:00"))
	insertPhoto(t, db, 6, nil)
	insertPhoto(t, db, 7, ptr("2023-12-31T23:59:59"))
	return db
}

func ids(ps []Photo) []int64 {
	out := []int64{}
	for _, p := range ps {
		out = append(out, p.ID)
	}
	return out
}

// cursorOf is the cursor a caller derives from the last row of a page.
func cursorOf(p Photo) *ListCursor {
	c := &ListCursor{ID: p.ID}
	if p.TakenAt != nil {
		c.TakenAt = *p.TakenAt
	}
	return c
}

var newestFirst = []int64{5, 4, 2, 1, 7, 6, 3}

func TestListingIsNewestFirstWithIdBreakingTiesAndUndatedFilesLast(t *testing.T) {
	db := seedPhotos(t)
	got, err := ListPhotos(db, nil, 100)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(ids(got), newestFirst) {
		t.Errorf("ListPhotos order = %v, want %v", ids(got), newestFirst)
	}
	all, err := AllPhotos(db)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(ids(all), newestFirst) {
		t.Errorf("AllPhotos order = %v, want %v", ids(all), newestFirst)
	}
	if got[5].TakenAt != nil || got[0].TakenAt == nil {
		t.Errorf("taken dates: first %v, undated %v", got[0].TakenAt, got[5].TakenAt)
	}
}

func TestListingHonoursTheLimit(t *testing.T) {
	db := seedPhotos(t)
	got, err := ListPhotos(db, nil, 3)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(ids(got), []int64{5, 4, 2}) {
		t.Errorf("first 3 = %v", ids(got))
	}
}

func TestPagingWithTheLastRowAsCursorVisitsEveryFileOnceInListingOrder(t *testing.T) {
	for _, limit := range []int{1, 2, 3, 7, 10} {
		db := seedPhotos(t)
		var seen []int64
		var cur *ListCursor
		for pages := 0; pages < 20; pages++ {
			page, err := ListPhotos(db, cur, limit)
			if err != nil {
				t.Fatal(err)
			}
			if len(page) == 0 {
				break
			}
			if len(page) > limit {
				t.Fatalf("limit %d: page of %d", limit, len(page))
			}
			seen = append(seen, ids(page)...)
			cur = cursorOf(page[len(page)-1])
		}
		if !reflect.DeepEqual(seen, newestFirst) {
			t.Errorf("limit %d: paged %v, want %v", limit, seen, newestFirst)
		}
	}
}

func TestAnEmptyIndexListsNoFiles(t *testing.T) {
	db := openPhotosDB(t)
	got, err := ListPhotos(db, nil, 10)
	if err != nil || got == nil || len(got) != 0 {
		t.Errorf("ListPhotos = %v, %v", got, err)
	}
}

func TestPhotoByIDReturnsTheIndexedFileOrReportsItMissing(t *testing.T) {
	db := seedPhotos(t)
	p, ok, err := PhotoByID(db, 5)
	if err != nil || !ok {
		t.Fatalf("PhotoByID(5) = %v, %v, %v", p, ok, err)
	}
	if p.ID != 5 || p.RelPath != "p5.jpg" || p.MediaType != "image" || p.Size != 50 || p.MtimeNs != 500 ||
		p.TakenAt == nil || *p.TakenAt != "2024-05-01T10:00:00" {
		t.Errorf("photo = %+v", p)
	}
	if u, ok, err := PhotoByID(db, 3); err != nil || !ok || u.TakenAt != nil {
		t.Errorf("undated photo = %+v, %v, %v", u, ok, err)
	}
	if _, ok, err := PhotoByID(db, 99); err != nil || ok {
		t.Errorf("PhotoByID(99) ok=%v err=%v", ok, err)
	}
}
