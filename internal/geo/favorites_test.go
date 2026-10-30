package geo

import (
	"context"
	"errors"
	"math"
	"reflect"
	"testing"
	"time"

	"photocurator/internal/store"
)

func TestFavouritePlacesAreListedByNameAndKeepTheirPlaceParts(t *testing.T) {
	ctx := context.Background()
	db, err := store.OpenAppDB(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	now := time.Unix(1700000000, 0)

	empty, err := ListFavorites(ctx, db)
	if err != nil || empty == nil || len(empty) != 0 {
		t.Fatalf("empty list = %#v, err = %v", empty, err)
	}

	parts := Parts{Sublocation: "Altstadt", City: "Wien", State: "Wien", Country: "Österreich", CountryCode: "at"}
	zoo, err := CreateFavorite(ctx, db, Favorite{Name: "  Zoo ", Lat: 48.18, Lon: 16.3, Parts: parts}, now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := CreateFavorite(ctx, db, Favorite{Name: "alpenhütte", Lat: 47.1, Lon: 11.2}, now); err != nil {
		t.Fatal(err)
	}
	if zoo.Name != "Zoo" || zoo.Parts.CountryCode != "AT" {
		t.Fatalf("created = %#v", zoo)
	}

	list, err := ListFavorites(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 || list[0].Name != "alpenhütte" || list[1].Name != "Zoo" {
		t.Fatalf("list = %#v", list)
	}
	if !reflect.DeepEqual(list[1], zoo) {
		t.Fatalf("stored %#v, created %#v", list[1], zoo)
	}

	renamed, err := RenameFavorite(ctx, db, zoo.ID, "Tiergarten", now)
	if err != nil {
		t.Fatal(err)
	}
	if renamed.Name != "Tiergarten" || renamed.Lat != zoo.Lat || renamed.Parts != zoo.Parts {
		t.Fatalf("renamed = %#v", renamed)
	}
	list, _ = ListFavorites(ctx, db)
	if list[0].Name != "alpenhütte" || list[1].Name != "Tiergarten" {
		t.Fatalf("list after rename = %#v", list)
	}
}

func TestAFavouriteNameDifferingOnlyInCaseAndSpacesIsADuplicate(t *testing.T) {
	ctx := context.Background()
	db, err := store.OpenAppDB(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	now := time.Unix(1700000000, 0)

	home, err := CreateFavorite(ctx, db, Favorite{Name: "Mein  Zuhause", Lat: 1, Lon: 2}, now)
	if err != nil {
		t.Fatal(err)
	}
	other, err := CreateFavorite(ctx, db, Favorite{Name: "Büro", Lat: 3, Lon: 4}, now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := CreateFavorite(ctx, db, Favorite{Name: " mein zuhause ", Lat: 5, Lon: 6}, now); !errors.Is(err, ErrFavoriteExists) {
		t.Fatalf("create duplicate err = %v", err)
	}
	if _, err := RenameFavorite(ctx, db, other.ID, "MEIN ZUHAUSE", now); !errors.Is(err, ErrFavoriteExists) {
		t.Fatalf("rename onto other err = %v", err)
	}
	// Changing only the spelling of its own key is not a clash.
	got, err := RenameFavorite(ctx, db, home.ID, "mein zuhause", now)
	if err != nil || got.Name != "mein zuhause" {
		t.Fatalf("own-key rename = %#v, err = %v", got, err)
	}
	list, _ := ListFavorites(ctx, db)
	if len(list) != 2 {
		t.Fatalf("list = %#v", list)
	}
}

func TestAVanishedFavouritePlaceCannotBeRenamedOrDeletedAndItsIDIsNeverReused(t *testing.T) {
	ctx := context.Background()
	db, err := store.OpenAppDB(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	now := time.Unix(1700000000, 0)

	a, err := CreateFavorite(ctx, db, Favorite{Name: "A", Lat: 1, Lon: 1}, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := DeleteFavorite(ctx, db, a.ID); err != nil {
		t.Fatal(err)
	}
	if err := DeleteFavorite(ctx, db, a.ID); !errors.Is(err, ErrFavoriteNotFound) {
		t.Fatalf("second delete err = %v", err)
	}
	if _, err := RenameFavorite(ctx, db, a.ID, "B", now); !errors.Is(err, ErrFavoriteNotFound) {
		t.Fatalf("rename err = %v", err)
	}
	b, err := CreateFavorite(ctx, db, Favorite{Name: "A", Lat: 1, Lon: 1}, now)
	if err != nil {
		t.Fatal(err)
	}
	if b.ID == a.ID {
		t.Fatalf("id %d was reused", b.ID)
	}
	if _, err := RenameFavorite(ctx, db, a.ID, "C", now); !errors.Is(err, ErrFavoriteNotFound) {
		t.Fatalf("rename of old id err = %v", err)
	}
}

func TestFavouritePlacesWithInvalidCoordinatesOrControlCharactersAreRefused(t *testing.T) {
	ctx := context.Background()
	db, err := store.OpenAppDB(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	now := time.Unix(1700000000, 0)

	good, err := CreateFavorite(ctx, db, Favorite{Name: "Gut", Lat: 1, Lon: 1}, now)
	if err != nil {
		t.Fatal(err)
	}
	bad := []Favorite{
		{Name: "x", Lat: 91, Lon: 0},
		{Name: "x", Lat: 0, Lon: -181},
		{Name: "x", Lat: math.NaN(), Lon: 0},
		{Name: "   ", Lat: 0, Lon: 0},
		{Name: "a\nb", Lat: 0, Lon: 0},
		{Name: "x", Lat: 0, Lon: 0, Parts: Parts{City: "Wi\x00en"}},
	}
	for _, f := range bad {
		_, err := CreateFavorite(ctx, db, f, now)
		var inErr *LocationInputError
		if !errors.As(err, &inErr) {
			t.Errorf("create %#v: err = %v, want LocationInputError", f, err)
		}
	}
	for _, name := range []string{"", "a\tb"} {
		_, err := RenameFavorite(ctx, db, good.ID, name, now)
		var inErr *LocationInputError
		if !errors.As(err, &inErr) {
			t.Errorf("rename %q: err = %v, want LocationInputError", name, err)
		}
	}
	list, _ := ListFavorites(ctx, db)
	if len(list) != 1 || list[0].Name != "Gut" {
		t.Fatalf("list = %#v", list)
	}
}
