package library

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

// selectionSeed: ids 1..2000 spread over July 2024 (ids rise with time), plus the two border
// photos 2001 (2024-06-30T23:59:59) and 2002 (2024-08-01T00:00:00).
func selectionSeed() []seed {
	base := time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC)
	var rows []seed
	for i := 1; i <= 2000; i++ {
		at := base.Add(time.Duration(i-1) * 1339 * time.Second) // 2000*1339s is about 31 days
		rows = append(rows, seed{int64(i), "2024/07", "image", at.Format("2006-01-02T15:04:05")})
	}
	rows = append(rows,
		seed{2001, "2024/06", "image", "2024-06-30T23:59:59"},
		seed{2002, "2024/08", "image", "2024-08-01T00:00:00"})
	return rows
}

func resolvedIDs(rs []Resolved) []int64 {
	out := []int64{}
	for _, r := range rs {
		out = append(out, r.ID)
	}
	return out
}

func TestSelection(t *testing.T) {
	ctx := context.Background()
	db := seedGrid(t, selectionSeed())
	resolve := func(t *testing.T, s Selection) []int64 {
		t.Helper()
		rs, err := ResolveSelection(ctx, db, s)
		if err != nil {
			t.Fatal(err)
		}
		return resolvedIDs(rs)
	}
	var day string
	if err := db.QueryRow(`SELECT substr(taken_at,1,10) FROM photos WHERE id = 1000`).Scan(&day); err != nil {
		t.Fatal(err)
	}

	t.Run("a whole month of 2000 photos minus three exceptions resolves to exactly those files", func(t *testing.T) {
		got := resolve(t, Selection{
			Filter:     SelectionFilter{Groups: []string{"2024-07"}},
			ExcludeIDs: []int64{5, 1000, 2000},
		})
		if len(got) != 1997 {
			t.Fatalf("got %d files, want 1997", len(got))
		}
		seen := map[int64]bool{}
		for i, id := range got {
			if id < 1 || id > 2000 || id == 5 || id == 1000 || id == 2000 || seen[id] {
				t.Fatalf("unexpected or duplicate id %d", id)
			}
			seen[id] = true
			if i > 0 && got[i-1] < id {
				t.Fatalf("not in grid order at %d", i)
			}
		}
		rs, err := ResolveSelection(ctx, db, Selection{Filter: SelectionFilter{Groups: []string{"2024-07"}}, ExcludeIDs: []int64{2000}})
		if err != nil || rs[0].ID != 1999 || rs[0].RelPath != "2024/07/p1999.jpg" {
			t.Errorf("first = %+v, %v", rs[0], err)
		}
	})

	t.Run("a day group and an include id from another month add up without duplicates", func(t *testing.T) {
		inDay := resolve(t, Selection{Filter: SelectionFilter{Groups: []string{day}}})
		if len(inDay) < 2 {
			t.Fatal("day group is too small")
		}
		got := resolve(t, Selection{
			Filter:     SelectionFilter{Groups: []string{day}},
			IncludeIDs: []int64{2001, inDay[0], 2001},
		})
		if len(got) != len(inDay)+1 {
			t.Fatalf("got %d, want %d", len(got), len(inDay)+1)
		}
		if got[len(got)-1] != 2001 {
			t.Errorf("the June photo should sort last, got %d", got[len(got)-1])
		}
	})

	t.Run("a range between two ids covers every photo between them in grid order, whichever end comes first", func(t *testing.T) {
		a := resolve(t, Selection{Filter: SelectionFilter{Ranges: [][2]int64{{100, 400}}}})
		b := resolve(t, Selection{Filter: SelectionFilter{Ranges: [][2]int64{{400, 100}}}})
		want := []int64{}
		for id := int64(400); id >= 100; id-- {
			want = append(want, id)
		}
		if !reflect.DeepEqual(a, want) || !reflect.DeepEqual(b, want) {
			t.Errorf("range mismatch: %d / %d files, want %d", len(a), len(b), len(want))
		}
		// across the month borders: the oldest to the newest photo is everything
		if all := resolve(t, Selection{Filter: SelectionFilter{Ranges: [][2]int64{{2002, 2001}}}}); len(all) != 2002 {
			t.Errorf("full range = %d files, want 2002", len(all))
		}
	})

	t.Run("an empty selection resolves to nothing", func(t *testing.T) {
		for _, s := range []Selection{{}, {ExcludeIDs: []int64{1}}, {Filter: SelectionFilter{Groups: []string{}}}} {
			if got := resolve(t, s); len(got) != 0 {
				t.Errorf("%+v resolved to %d files", s, len(got))
			}
			if n, err := CountSelection(ctx, db, s); err != nil || n != 0 {
				t.Errorf("count = %d, %v", n, err)
			}
		}
	})

	t.Run("an excluded id wins over its group and over include_ids", func(t *testing.T) {
		got := resolve(t, Selection{
			Filter:     SelectionFilter{Groups: []string{"2024-06"}},
			IncludeIDs: []int64{2001, 7},
			ExcludeIDs: []int64{2001},
		})
		if !reflect.DeepEqual(got, []int64{7}) {
			t.Errorf("got %v, want [7]", got)
		}
	})

	t.Run("count and resolve agree", func(t *testing.T) {
		for i, s := range []Selection{
			{Filter: SelectionFilter{Groups: []string{"2024-07", "2024-08", day}}, ExcludeIDs: []int64{1, 2}},
			{Filter: SelectionFilter{Ranges: [][2]int64{{10, 20}, {15, 30}}}, IncludeIDs: []int64{2002}},
			{IncludeIDs: []int64{5, 6, 99999}},
			{},
		} {
			rs := resolve(t, s)
			n, err := CountSelection(ctx, db, s)
			if err != nil || n != len(rs) {
				t.Errorf("case %d: count = %d, %v; resolve = %d", i, n, err, len(rs))
			}
		}
	})

	t.Run("malformed group keys, unknown range ids and oversized lists are rejected", func(t *testing.T) {
		big := make([]int64, maxSelectionIDs+1)
		many := make([]string, maxSelectionFilters+1)
		for i := range many {
			many[i] = "2024-07"
		}
		bad := []Selection{
			{Filter: SelectionFilter{Groups: []string{"2024-7"}}},
			{Filter: SelectionFilter{Groups: []string{"2024-13"}}},
			{Filter: SelectionFilter{Groups: []string{"2024-07-32"}}},
			{Filter: SelectionFilter{Groups: []string{"2024"}}},
			{Filter: SelectionFilter{Groups: []string{"2024-07'; DROP TABLE photos;--"}}},
			{Filter: SelectionFilter{Ranges: [][2]int64{{1, 424242}}}},
			{IncludeIDs: big},
			{ExcludeIDs: big},
			{Filter: SelectionFilter{Groups: many}},
		}
		for i, s := range bad {
			if _, err := ResolveSelection(ctx, db, s); !errors.Is(err, ErrInvalidSelection) {
				t.Errorf("case %d resolve: err = %v", i, err)
			}
			if _, err := CountSelection(ctx, db, s); !errors.Is(err, ErrInvalidSelection) {
				t.Errorf("case %d count: err = %v", i, err)
			}
		}
		if err := (Selection{IncludeIDs: big[:maxSelectionIDs]}).Validate(); err != nil {
			t.Errorf("a list at the limit must pass: %v", err)
		}
	})
}
