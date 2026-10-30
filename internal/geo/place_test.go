package geo

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "nominatim", name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestAPOIHitBecomesNameAndTownWithItsPlaceParts(t *testing.T) {
	places, err := ParseSearch(fixture(t, "search-ritzenhofer-saalfelden.json"))
	if err != nil {
		t.Fatal(err)
	}
	p := places[0]
	if p.Label != "Ritzenhofer, Saalfelden" || p.Name != "Ritzenhofer" || p.Kind != "hotel" {
		t.Errorf("got label %q name %q kind %q", p.Label, p.Name, p.Kind)
	}
	want := Parts{Sublocation: "Ritzenhofer", City: "Saalfelden", State: "Salzburg", Country: "Österreich", CountryCode: "AT"}
	if p.Parts != want {
		t.Errorf("parts = %+v, want %+v", p.Parts, want)
	}
	if p.Lat < 47.42 || p.Lat > 47.43 || p.Lon < 12.84 || p.Lon > 12.85 {
		t.Errorf("coordinates = %v, %v", p.Lat, p.Lon)
	}
}

func TestAHitWithoutItsOwnNameIsLabelledWithTheTownAlone(t *testing.T) {
	places, err := ParseSearch(fixture(t, "search-ritzenhofer-saalfelden.json"))
	if err != nil {
		t.Fatal(err)
	}
	if got := places[1].Label; got != "Saalfelden" {
		t.Errorf("nameless hit label = %q", got)
	}

	town, err := ParseSearch(fixture(t, "search-saalfelden.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(town) != 1 || town[0].Label != "Saalfelden" || town[0].Parts.City != "Saalfelden" {
		t.Errorf("town hit = %+v", town)
	}
}

func TestSearchResultsKeepNominatimsOrderAndStopAtTen(t *testing.T) {
	places, err := ParseSearch(fixture(t, "search-ritzenhofer-saalfelden.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(places) != 10 {
		t.Fatalf("got %d places, want 10", len(places))
	}
	if places[0].Name != "Ritzenhofer" || places[1].Name != "" {
		t.Errorf("first two = %q, %q", places[0].Name, places[1].Name)
	}
	for i := 2; i < 10; i++ {
		if want := fmt.Sprintf("Ritzen Haus %d", i+1); places[i].Name != want {
			t.Errorf("places[%d] = %q, want %q", i, places[i].Name, want)
		}
	}
}

func TestReverseAnswerBecomesAPlaceAndNoResultBecomesNil(t *testing.T) {
	p, err := ParseReverse(fixture(t, "reverse-saalfelden.json"))
	if err != nil || p == nil {
		t.Fatalf("p = %v, err = %v", p, err)
	}
	if p.Label != "Gasthof Beispiel, Saalfelden" || p.Kind != "restaurant" {
		t.Errorf("got %+v", p)
	}

	p, err = ParseReverse([]byte(`{"error":"Unable to geocode"}`))
	if p != nil || err != nil {
		t.Errorf("p = %v, err = %v", p, err)
	}
}
