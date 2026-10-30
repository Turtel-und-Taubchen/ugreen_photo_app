package tags

import (
	"context"
	"reflect"
	"testing"

	"photocurator/internal/store"
)

func TestTheMostRecentlyUsedTagComesFirstAndOnlyEightAreKept(t *testing.T) {
	ctx := context.Background()
	db, err := store.OpenAppDB(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if got, err := ListRecent(ctx, db); err != nil || got == nil || len(got) != 0 {
		t.Fatalf("empty list = %#v, err = %v", got, err)
	}
	all := []string{"t0", "t1", "t2", "t3", "t4", "t5", "t6", "t7", "t8", "t9"}
	for _, p := range all {
		if err := TouchRecent(ctx, db, []string{p}); err != nil {
			t.Fatal(err)
		}
	}
	got, _ := ListRecent(ctx, db)
	if want := []string{"t9", "t8", "t7", "t6", "t5", "t4", "t3", "t2"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("list = %v", got)
	}
	if err := TouchRecent(ctx, db, []string{"t4"}); err != nil {
		t.Fatal(err)
	}
	got, _ = ListRecent(ctx, db)
	if want := []string{"t4", "t9", "t8", "t7", "t6", "t5", "t3", "t2"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("list after retouch = %v", got)
	}
	if err := TouchRecent(ctx, db, []string{"A", "B"}); err != nil {
		t.Fatal(err)
	}
	got, _ = ListRecent(ctx, db)
	if want := []string{"B", "A", "t4", "t9", "t8", "t7", "t6", "t5"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("list after batch = %v", got)
	}
}
