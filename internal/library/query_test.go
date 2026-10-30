package library

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"testing"
)

type seed struct {
	id                  int64
	folder, media, when string
}

func seedGrid(t *testing.T, rows []seed) *sql.DB {
	t.Helper()
	db := openPhotosDB(t)
	for _, r := range rows {
		var at any
		if r.when != "" {
			at = r.when
		}
		_, err := db.Exec(`INSERT INTO photos (id, rel_path, folder, media_type, format, size, mtime_ns, taken_at, taken_source, scanned_at)
			VALUES (?, ?, ?, ?, 'jpeg', 1, 1, ?, 'exif', '2024-01-01T00:00:00Z')`,
			r.id, fmt.Sprintf("%s/p%d.jpg", r.folder, r.id), r.folder, r.media, at)
		if err != nil {
			t.Fatal(err)
		}
	}
	return db
}

var gridSeed = []seed{
	{1, "2024/07", "image", "2024-07-12T10:00:00"},
	{2, "2024/07", "video", "2024-07-12T09:00:00"},
	{3, "2024/07", "image", "2024-07-03T09:00:00"},
	{4, "2024/06", "image", "2024-06-30T23:00:00"},
	{5, "misc", "image", "2023-12-31T12:00:00"},
	{6, "2024/07", "image", ""},
}

func TestGroupsCountPhotosAndVideosPerMonthAndDayNewestFirst(t *testing.T) {
	db := seedGrid(t, gridSeed)
	g, err := GroupCounts(context.Background(), db)
	if err != nil {
		t.Fatal(err)
	}
	want := Groups{
		Total: 6,
		Months: []MonthGroup{
			{"2024-07", 3, []DayCount{{"2024-07-12", 2}, {"2024-07-03", 1}}},
			{"2024-06", 1, []DayCount{{"2024-06-30", 1}}},
			{"2023-12", 1, []DayCount{{"2023-12-31", 1}}},
		},
		Folders: []FolderCount{{"2024/07", 4}, {"2024/06", 1}},
	}
	if !reflect.DeepEqual(g, want) {
		t.Errorf("groups = %+v\nwant     %+v", g, want)
	}
}

func TestPagesFollowTheKeysetCursorWithoutGapsOrRepeats(t *testing.T) {
	// ids 3..5 share one taken date, so the id has to break the tie.
	db := seedGrid(t, []seed{
		{1, "2024/07", "image", "2024-07-01T10:00:00"},
		{2, "2024/07", "image", "2024-07-02T10:00:00"},
		{3, "2024/07", "image", "2024-07-10T10:00:00"},
		{4, "2024/07", "image", "2024-07-10T10:00:00"},
		{5, "2024/07", "image", "2024-07-10T10:00:00"},
		{6, "2024/07", "image", "2024-07-06T10:00:00"},
		{7, "2024/07", "image", "2024-07-07T10:00:00"},
		{8, "2024/08", "image", "2024-08-01T00:00:00"},
		{9, "2024/06", "image", "2024-06-30T23:59:59"},
	})
	for _, limit := range []int{1, 2, 3, 7} {
		var seen []int64
		var cur *ListCursor
		for i := 0; i < 20; i++ {
			page, err := ListMonth(context.Background(), db, "2024-07", cur, limit)
			if err != nil {
				t.Fatal(err)
			}
			if len(page) == 0 {
				break
			}
			seen = append(seen, ids(page)...)
			cur = cursorOf(page[len(page)-1])
		}
		if want := []int64{5, 4, 3, 7, 6, 2, 1}; !reflect.DeepEqual(seen, want) {
			t.Errorf("limit %d: paged %v, want %v", limit, seen, want)
		}
	}
}

func TestTheDetailCarriesTagsPlaceReviewStateDateSourceAndPath(t *testing.T) {
	db := seedGrid(t, gridSeed)
	_, err := db.Exec(`UPDATE photos SET taken_source = 'filename', loc_label = 'Rathaus, Wien', loc_city = 'Wien',
		loc_country = 'Austria', review_state = 'done', read_error = 'boom', format = 'heic' WHERE id = 3`)
	if err != nil {
		t.Fatal(err)
	}
	for _, tag := range []string{"Person/Anna", "Anlass/Geburtstag"} {
		if _, err := db.Exec(`INSERT INTO photo_tags VALUES (3, ?)`, tag); err != nil {
			t.Fatal(err)
		}
	}
	d, ok, err := PhotoDetail(context.Background(), db, 3)
	if err != nil || !ok {
		t.Fatalf("PhotoDetail = %v, %v", ok, err)
	}
	if d.RelPath != "2024/07/p3.jpg" || d.Folder != "2024/07" || d.MediaType != "image" || d.Format != "heic" ||
		d.TakenSource != "filename" || d.ReviewState != "done" || d.ReadError == nil || *d.ReadError != "boom" ||
		*d.TakenAt != "2024-07-03T09:00:00" {
		t.Errorf("detail = %+v", d)
	}
	if !reflect.DeepEqual(d.Tags, []string{"Anlass/Geburtstag", "Person/Anna"}) {
		t.Errorf("tags = %v", d.Tags)
	}
	if want := (Place{Label: "Rathaus, Wien", City: "Wien", Country: "Austria"}); d.Place != want {
		t.Errorf("place = %+v", d.Place)
	}
	if _, ok, err := PhotoDetail(context.Background(), db, 99); ok || err != nil {
		t.Errorf("unknown id: ok=%v err=%v", ok, err)
	}
	if bare, _, _ := PhotoDetail(context.Background(), db, 1); bare.Tags == nil || len(bare.Tags) != 0 {
		t.Errorf("untagged tags = %#v", bare.Tags)
	}
}

func TestPreviousAndNextFollowTheGridOrderAcrossMonthsAndStopAtTheEnds(t *testing.T) {
	db := seedGrid(t, gridSeed)
	// Grid order, newest first: 1, 2, 3, 4, 5, then the undated 6.
	order := []int64{1, 2, 3, 4, 5, 6}
	for i, id := range order {
		d, ok, err := PhotoDetail(context.Background(), db, id)
		if err != nil || !ok {
			t.Fatalf("id %d: %v %v", id, ok, err)
		}
		var prev, next *int64
		if i > 0 {
			prev = &order[i-1]
		}
		if i < len(order)-1 {
			next = &order[i+1]
		}
		if !reflect.DeepEqual(d.PrevID, prev) || !reflect.DeepEqual(d.NextID, next) {
			t.Errorf("id %d: prev %v next %v, want %v %v", id, deref(d.PrevID), deref(d.NextID), deref(prev), deref(next))
		}
	}
}

func TestPreviousAndNextBreakTiesBetweenEqualTakenAtById(t *testing.T) {
	db := seedGrid(t, []seed{
		{1, "2024/07", "image", "2024-07-12T10:00:00"},
		{2, "2024/07", "image", "2024-07-12T09:00:00"},
		{3, "2024/07", "image", "2024-07-12T09:00:00"},
		{4, "2024/07", "image", "2024-07-12T09:00:00"},
		{5, "2024/07", "image", "2024-07-12T08:00:00"},
	})
	listed, err := ListMonth(context.Background(), db, "2024-07", nil, 10)
	if err != nil {
		t.Fatal(err)
	}
	var order []int64
	for _, p := range listed {
		order = append(order, p.ID)
	}
	if want := []int64{1, 4, 3, 2, 5}; !reflect.DeepEqual(order, want) {
		t.Fatalf("ListMonth order = %v, want %v", order, want)
	}
	for i, id := range order {
		d, ok, err := PhotoDetail(context.Background(), db, id)
		if err != nil || !ok {
			t.Fatalf("id %d: %v %v", id, ok, err)
		}
		var prev, next *int64
		if i > 0 {
			prev = &order[i-1]
		}
		if i < len(order)-1 {
			next = &order[i+1]
		}
		if !reflect.DeepEqual(d.PrevID, prev) || !reflect.DeepEqual(d.NextID, next) {
			t.Errorf("id %d: prev %v next %v, want %v %v", id, deref(d.PrevID), deref(d.NextID), deref(prev), deref(next))
		}
	}
}

func deref(p *int64) any {
	if p == nil {
		return nil
	}
	return *p
}
