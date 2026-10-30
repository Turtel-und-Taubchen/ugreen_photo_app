package library

import (
	"context"
	"testing"
)

func TestSubtreeCountsTreatUnderscoreAndPercentLiterally(t *testing.T) {
	ctx := context.Background()
	db := seedGrid(t, []seed{
		{1, "f", "image", ""}, {2, "f", "image", ""}, {3, "f", "image", ""},
	})
	for id, p := range map[int]string{1: "A_b/x", 2: "Axb/y", 3: "A%/z"} {
		if _, err := db.Exec(`INSERT INTO photo_tags (photo_id, tag_path) VALUES (?, ?)`, id, p); err != nil {
			t.Fatal(err)
		}
	}
	for _, c := range []struct {
		path string
		want int
	}{{"A_b", 1}, {"A%", 1}} {
		got, err := TagSubtreeCount(ctx, db, c.path)
		if err != nil || got != c.want {
			t.Fatalf("count(%q) = %d, err = %v, want %d", c.path, got, err, c.want)
		}
	}
}
