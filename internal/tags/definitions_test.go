package tags

import (
	"context"
	"reflect"
	"testing"

	"photocurator/internal/store"
)

func TestCreatingAPathAlsoCreatesItsMissingParents(t *testing.T) {
	ctx := context.Background()
	db, err := store.OpenAppDB(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	created, err := Create(ctx, db, "Anlass/Geburtstag")
	if err != nil || !created {
		t.Fatalf("created = %v, err = %v", created, err)
	}
	got, _ := List(ctx, db)
	if want := []string{"Anlass", "Anlass/Geburtstag"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("list = %v", got)
	}
	created, err = Create(ctx, db, "Anlass/Geburtstag")
	if err != nil || created {
		t.Fatalf("second create: created = %v, err = %v", created, err)
	}
}

func TestDeletingACategoryRemovesItsEmptyDescendantsButNotItsParentOrALookalike(t *testing.T) {
	ctx := context.Background()
	db, err := store.OpenAppDB(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, p := range []string{"A", "A/B", "A/B/C", "A_B"} {
		if _, err := Create(ctx, db, p); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := DeleteSubtree(ctx, db, "A/B"); err != nil {
		t.Fatal(err)
	}
	got, _ := List(ctx, db)
	if want := []string{"A", "A_B"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("list = %v", got)
	}
}
