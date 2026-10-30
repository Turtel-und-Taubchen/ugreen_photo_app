package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"photocurator/internal/api"
	"photocurator/internal/geo"
	"photocurator/internal/store"
)

func placesServer(t *testing.T) *testServer {
	t.Helper()
	db, err := store.OpenAppDB(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return newTestServerDeps(t, api.LibraryDeps{Places: db})
}

func listPlaces(t *testing.T, s *testServer, sess reqOpt) []geo.Favorite {
	t.Helper()
	rec := s.do(http.MethodGet, "/api/places", "", sess)
	wantJSONStatus(t, "list places", rec, http.StatusOK)
	var got struct {
		Places []geo.Favorite `json:"places"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil || got.Places == nil {
		t.Fatalf("places must be an array, body %s", rec.Body.String())
	}
	return got.Places
}

func placeOf(t *testing.T, rec *httptest.ResponseRecorder) geo.Favorite {
	t.Helper()
	var got struct {
		Place geo.Favorite `json:"place"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	return got.Place
}

func errorOf(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var e struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &e); err != nil {
		t.Fatal(err)
	}
	return e.Error
}

func TestFavouritePlacesAreOneListThatBothUsersMaintain(t *testing.T) {
	s := placesServer(t)
	anna, ben := s.signedIn(t, "anna"), s.signedIn(t, "ben")

	if got := listPlaces(t, s, anna); len(got) != 0 {
		t.Fatalf("fresh list = %+v, want empty", got)
	}

	rec := s.do(http.MethodPost, "/api/places",
		`{"name":"Zuhause","lat":48.2,"lon":16.3,"parts":{"city":"Wien","country":"Österreich","countryCode":"AT"}}`, anna, withXRW)
	wantJSONStatus(t, "create", rec, http.StatusCreated)
	created := placeOf(t, rec)
	if created.ID <= 0 || created.Name != "Zuhause" || created.Lat != 48.2 || created.Lon != 16.3 || created.Parts.City != "Wien" {
		t.Fatalf("created = %+v", created)
	}

	seen := listPlaces(t, s, ben)
	if len(seen) != 1 || seen[0] != created {
		t.Fatalf("ben sees %+v, want [%+v]", seen, created)
	}

	path := "/api/places/" + strconv.FormatInt(created.ID, 10)
	rec = s.do(http.MethodPatch, path, `{"name":"Daheim"}`, ben, withXRW)
	wantJSONStatus(t, "rename", rec, http.StatusOK)
	if got := placeOf(t, rec); got.Name != "Daheim" || got.ID != created.ID || got.Lat != created.Lat {
		t.Fatalf("renamed = %+v", got)
	}
	if seen := listPlaces(t, s, anna); len(seen) != 1 || seen[0].Name != "Daheim" {
		t.Fatalf("anna sees %+v, want the new name", seen)
	}

	if rec := s.do(http.MethodDelete, path, "", ben, withXRW); rec.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, want 204", rec.Code)
	}
	if got := listPlaces(t, s, anna); len(got) != 0 {
		t.Fatalf("list after delete = %+v, want empty", got)
	}
}

func TestFavouritePlaceRequestsWithBadInputDuplicatesOrUnknownIdsAreRefused(t *testing.T) {
	s := placesServer(t)
	sess := s.signedIn(t, "anna")

	rec := s.do(http.MethodPost, "/api/places", `{"name":"Zuhause","lat":48.2,"lon":16.3,"parts":{}}`, sess, withXRW)
	wantJSONStatus(t, "create", rec, http.StatusCreated)
	id := strconv.FormatInt(placeOf(t, rec).ID, 10)

	for _, c := range []struct{ label, method, target, body string }{
		{"broken json", http.MethodPost, "/api/places", `{"name":`},
		{"empty name", http.MethodPost, "/api/places", `{"name":"  ","lat":1,"lon":1,"parts":{}}`},
		{"latitude out of range", http.MethodPost, "/api/places", `{"name":"Nord","lat":91,"lon":1,"parts":{}}`},
		{"broken rename json", http.MethodPatch, "/api/places/" + id, `nope`},
		{"empty rename", http.MethodPatch, "/api/places/" + id, `{"name":""}`},
		{"id not a number", http.MethodPatch, "/api/places/abc", `{"name":"X"}`},
		{"id zero", http.MethodDelete, "/api/places/0", ""},
		{"id negative", http.MethodDelete, "/api/places/-3", ""},
	} {
		wantJSONStatus(t, c.label, s.do(c.method, c.target, c.body, sess, withXRW), http.StatusBadRequest)
	}
	rec = s.do(http.MethodPost, "/api/places", `{"name":`, sess, withXRW)
	if got := errorOf(t, rec); got != "Ungültige Anfrage." {
		t.Errorf("broken json error = %q", got)
	}

	rec = s.do(http.MethodPost, "/api/places", `{"name":"ZUHAUSE","lat":1,"lon":1,"parts":{}}`, sess, withXRW)
	wantJSONStatus(t, "duplicate", rec, http.StatusConflict)
	if got, want := errorOf(t, rec), "Es gibt schon einen Lieblingsort „ZUHAUSE“."; got != want {
		t.Errorf("duplicate error = %q, want %q", got, want)
	}

	other := s.do(http.MethodPost, "/api/places", `{"name":"Büro","lat":1,"lon":1,"parts":{}}`, sess, withXRW)
	wantJSONStatus(t, "second create", other, http.StatusCreated)
	rec = s.do(http.MethodPatch, "/api/places/"+id, `{"name":"Büro"}`, sess, withXRW)
	wantJSONStatus(t, "rename onto existing name", rec, http.StatusConflict)
	if got, want := errorOf(t, rec), "Es gibt schon einen Lieblingsort „Büro“."; got != want {
		t.Errorf("error = %q, want %q", got, want)
	}

	for _, c := range []struct{ method, body string }{
		{http.MethodPatch, `{"name":"X"}`},
		{http.MethodDelete, ""},
	} {
		rec := s.do(c.method, "/api/places/9999", c.body, sess, withXRW)
		wantJSONStatus(t, c.method+" unknown id", rec, http.StatusNotFound)
		if got := errorOf(t, rec); got != "Lieblingsort nicht gefunden." {
			t.Errorf("%s error = %q", c.method, got)
		}
	}

	if got := listPlaces(t, s, sess); len(got) != 2 {
		t.Errorf("list = %+v, refused requests must change nothing", got)
	}
}
