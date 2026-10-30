package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"photocurator/internal/api"
	"photocurator/internal/library"
)

type fakeReader struct{ fakeLister }

func (fakeReader) Groups(context.Context) (library.Groups, error) {
	return library.Groups{Total: 3, Months: []library.MonthGroup{{Month: "2024-07", Count: 2,
		Days: []library.DayCount{{Day: "2024-07-12", Count: 2}}}}, Folders: []library.FolderCount{{Folder: "2024/07", Count: 2}}}, nil
}

func (fakeReader) Detail(_ context.Context, id int64) (library.Detail, bool, error) {
	if id != 7 {
		return library.Detail{}, false, nil
	}
	at, next := "2024-07-12T10:00:00", int64(5)
	return library.Detail{Photo: library.Photo{ID: 7, RelPath: "2024/07/a.jpg", MediaType: "image", TakenAt: &at},
		Folder: "2024/07", Format: "jpeg", TakenSource: "exif", Tags: []string{"Person/Anna"},
		Place: library.Place{City: "Wien"}, ReviewState: "open", NextID: &next}, true, nil
}

type coordReader struct{ fakeReader }

func (coordReader) Detail(_ context.Context, id int64) (library.Detail, bool, error) {
	lat, lon := 48.2, 16.37
	d := library.Detail{Photo: library.Photo{ID: id, RelPath: "a.jpg", MediaType: "image"}, Tags: []string{}}
	if id == 1 {
		d.Place = library.Place{Lat: &lat, Lon: &lon}
	}
	return d, true, nil
}

func TestPhotoDetailIncludesCoordinates(t *testing.T) {
	s := newTestServerDeps(t, api.LibraryDeps{Photos: coordReader{}})
	sess := s.signedIn(t, "anna")
	for id, wantSet := range map[string]bool{"1": true, "2": false} {
		rec := s.do(http.MethodGet, "/api/photos/"+id, "", sess)
		wantJSONStatus(t, id, rec, http.StatusOK)
		var d struct {
			Location map[string]any `json:"location"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &d); err != nil {
			t.Fatal(err)
		}
		lat, hasLat := d.Location["lat"]
		lon, hasLon := d.Location["lon"]
		if !hasLat || !hasLon {
			t.Fatalf("photo %s: location lacks lat/lon keys: %v", id, d.Location)
		}
		if wantSet && (lat != 48.2 || lon != 16.37) {
			t.Errorf("photo %s: lat/lon = %v/%v, want 48.2/16.37", id, lat, lon)
		}
		if !wantSet && (lat != nil || lon != nil) {
			t.Errorf("photo %s: lat/lon = %v/%v, want null", id, lat, lon)
		}
	}
}

func TestPhotoReadRoutesAnswerWithTheDocumentedShapeAndRejectBadParameters(t *testing.T) {
	s := newTestServerDeps(t, api.LibraryDeps{Photos: fakeReader{}})
	sess := s.signedIn(t, "anna")

	for _, target := range []string{"/api/photos", "/api/photos/groups", "/api/photos/7"} {
		wantJSONStatus(t, target, s.do(http.MethodGet, target, ""), http.StatusUnauthorized)
	}

	rec := s.do(http.MethodGet, "/api/photos/groups", "", sess)
	wantJSONStatus(t, "groups", rec, http.StatusOK)
	var g map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &g); err != nil {
		t.Fatal(err)
	}
	m := g["months"].([]any)[0].(map[string]any)
	if g["total"] != 3.0 || m["month"] != "2024-07" || m["count"] != 2.0 ||
		m["days"].([]any)[0].(map[string]any)["day"] != "2024-07-12" ||
		g["folders"].([]any)[0].(map[string]any)["folder"] != "2024/07" {
		t.Errorf("groups = %v", g)
	}

	rec = s.do(http.MethodGet, "/api/photos/7", "", sess)
	wantJSONStatus(t, "detail", rec, http.StatusOK)
	var d map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &d); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"id", "rel_path", "folder", "media_type", "format", "taken_at", "taken_source", "tags",
		"location", "review_state", "read_error", "prev_id", "next_id", "thumb_key"} {
		if _, ok := d[k]; !ok {
			t.Errorf("detail lacks %q: %v", k, d)
		}
	}
	if d["prev_id"] != nil || d["next_id"] != 5.0 || d["location"].(map[string]any)["city"] != "Wien" {
		t.Errorf("detail = %v", d)
	}

	for target, want := range map[string]int{
		"/api/photos/99":                   http.StatusNotFound,
		"/api/photos/abc":                  http.StatusNotFound,
		"/api/photos?month=2024-13":        http.StatusBadRequest,
		"/api/photos?month=July":           http.StatusBadRequest,
		"/api/photos?month=2024-07":        http.StatusOK,
		"/api/photos?limit=0":              http.StatusBadRequest,
		"/api/photos?limit=x":              http.StatusBadRequest,
		"/api/photos?limit=9999":           http.StatusOK,
		"/api/photos?month=2024-07&after=": http.StatusOK,
	} {
		wantJSONStatus(t, target, s.do(http.MethodGet, target, "", sess), want)
	}
}
