package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"

	"photocurator/internal/api"
	"photocurator/internal/store"
	"photocurator/internal/writer"
)

func jobsServer(t *testing.T) (*testServer, *writer.Queue) {
	t.Helper()
	db, err := store.OpenAppDB(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	q := &writer.Queue{DB: db}
	idx := selectionIndex(t)
	return newTestServerDeps(t, api.LibraryDeps{Index: idx, Writes: q, Tags: &api.TagDeps{App: db, Index: idx}}), q
}

func TestCreateJobResolvesTheSelectionAndQueuesIt(t *testing.T) {
	s, q := jobsServer(t)
	sess := s.signedIn(t, "anna")
	body := `{"selection":{"filter":{"groups":["2024-07"]},"exclude_ids":[2]},"action":"tags.add","payload":{"tags":[" Anlass / Geburtstag "]}}`

	wantJSONStatus(t, "no session", s.do(http.MethodPost, "/api/jobs", body, withXRW), http.StatusUnauthorized)

	rec := s.do(http.MethodPost, "/api/jobs", body, sess, withXRW)
	wantJSONStatus(t, "create", rec, http.StatusAccepted)
	var created struct{ ID int64 }
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil || created.ID == 0 {
		t.Fatalf("body %q has no id", rec.Body.String())
	}

	job, items, err := q.Get(context.Background(), created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if job.Username != "anna" || job.Action != "tags.add" || job.Status != writer.JobQueued || job.Total != 3 {
		t.Errorf("job = %+v, want anna's queued tags.add over 3 files", job)
	}
	var got []string
	for _, it := range items {
		got = append(got, it.RelPath)
	}
	sort.Strings(got)
	if want := []string{"p1.jpg", "p3.jpg", "p4.jpg"}; !reflect.DeepEqual(got, want) {
		t.Errorf("files = %v, want %v", got, want)
	}

	var payload string
	if err := q.DB.QueryRow(`SELECT payload FROM jobs WHERE id = ?`, created.ID).Scan(&payload); err != nil {
		t.Fatal(err)
	}
	if want := `{"AddTags":["Anlass/Geburtstag"],"RemoveTags":null,"Location":null,"ReviewState":""}`; payload != want {
		t.Errorf("payload = %s, want %s", payload, want)
	}
}

func TestCreateJobRejectsUnknownActionAndBadTags(t *testing.T) {
	s, q := jobsServer(t)
	sess := s.signedIn(t, "anna")
	sel := `"selection":{"include_ids":[1]}`
	for name, body := range map[string]string{
		"unknown action":  `{` + sel + `,"action":"tags.rename","payload":{"tags":["A"]}}`,
		"no tags":         `{` + sel + `,"action":"tags.add","payload":{"tags":[]}}`,
		"empty segment":   `{` + sel + `,"action":"tags.add","payload":{"tags":["A//B"]}}`,
		"separator":       `{` + sel + `,"action":"tags.add","payload":{"tags":["A|B"]}}`,
		"empty selection": `{"selection":{},"action":"tags.add","payload":{"tags":["A"]}}`,
		"bad month":       `{"selection":{"filter":{"groups":["2024-13"]}},"action":"tags.add","payload":{"tags":["A"]}}`,
		"not json":        `{"selection":`,
	} {
		rec := s.do(http.MethodPost, "/api/jobs", body, sess, withXRW)
		wantJSONStatus(t, name, rec, http.StatusBadRequest)
	}
	if jobs, err := q.List(context.Background(), 10); err != nil || len(jobs) != 0 {
		t.Errorf("jobs = %v, %v; a rejected request must queue nothing", jobs, err)
	}
}

func TestJobDetailListsEveryFailedFileWithReason(t *testing.T) {
	s, q := jobsServer(t)
	sess := s.signedIn(t, "anna")
	create := func(ids string) int64 {
		rec := s.do(http.MethodPost, "/api/jobs",
			`{"selection":{"include_ids":`+ids+`},"action":"tags.add","payload":{"tags":["A"]}}`, sess, withXRW)
		wantJSONStatus(t, "create", rec, http.StatusAccepted)
		var c struct{ ID int64 }
		if err := json.Unmarshal(rec.Body.Bytes(), &c); err != nil {
			t.Fatal(err)
		}
		return c.ID
	}
	first, second := create("[1,2,3]"), create("[4]")

	for rel, reason := range map[string]string{"p1.jpg": "Datei ist schreibgeschützt", "p3.jpg": "ExifTool: Fehler"} {
		if _, err := q.DB.Exec(`UPDATE job_items SET state = 'failed', reason = ? WHERE job_id = ? AND rel_path = ?`, reason, first, rel); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := q.DB.Exec(`UPDATE job_items SET state = 'ok' WHERE job_id = ? AND rel_path = 'p2.jpg'`, first); err != nil {
		t.Fatal(err)
	}
	if _, err := q.DB.Exec(`UPDATE jobs SET status = 'done', done = 3, ok = 1, failed = 2, finished_at = 1700000000 WHERE id = ?`, first); err != nil {
		t.Fatal(err)
	}

	rec := s.do(http.MethodGet, "/api/jobs/"+strconv.FormatInt(first, 10), "", sess, withXRW)
	wantJSONStatus(t, "detail", rec, http.StatusOK)
	var d struct {
		ID         int64   `json:"id"`
		Status     string  `json:"status"`
		Failed     int     `json:"failed"`
		FinishedAt *string `json:"finished_at"`
		Items      []struct {
			RelPath string `json:"rel_path"`
			State   string `json:"state"`
			Reason  string `json:"reason"`
		} `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &d); err != nil {
		t.Fatal(err)
	}
	if d.Status != "done" || d.Failed != 2 || d.FinishedAt == nil || len(d.Items) != 3 {
		t.Fatalf("detail = %+v (body %s)", d, rec.Body.String())
	}
	failed := map[string]string{}
	for _, it := range d.Items {
		if it.State == "failed" {
			failed[it.RelPath] = it.Reason
		}
	}
	want := map[string]string{"p1.jpg": "Datei ist schreibgeschützt", "p3.jpg": "ExifTool: Fehler"}
	if !reflect.DeepEqual(failed, want) {
		t.Errorf("failed items = %v, want %v", failed, want)
	}

	rec = s.do(http.MethodGet, "/api/jobs", "", sess, withXRW)
	wantJSONStatus(t, "list", rec, http.StatusOK)
	var list []struct {
		ID    int64            `json:"id"`
		Items *json.RawMessage `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 || list[0].ID != second || list[1].ID != first || list[0].Items != nil {
		t.Errorf("list = %s, want [%d, %d] newest first without items", rec.Body.String(), second, first)
	}

	wantJSONStatus(t, "unknown id", s.do(http.MethodGet, "/api/jobs/999", "", sess, withXRW), http.StatusNotFound)
}

// doneJob inserts a finished job with one ok item per path, as the runner leaves it.
func doneJob(t *testing.T, q *writer.Queue, user string, rels ...string) int64 {
	t.Helper()
	res, err := q.DB.Exec(`INSERT INTO jobs (username, action, payload, status, total, done, ok, created_at, finished_at)
		VALUES (?, 'tags.add', '{}', 'done', ?, ?, ?, 1700000000, 1700000001)`, user, len(rels), len(rels), len(rels))
	if err != nil {
		t.Fatal(err)
	}
	id, _ := res.LastInsertId()
	for _, r := range rels {
		if _, err := q.DB.Exec(`INSERT INTO job_items (job_id, rel_path, state, updated_at) VALUES (?, ?, 'ok', 1700000000)`, id, r); err != nil {
			t.Fatal(err)
		}
	}
	return id
}

func TestUndoEndpointQueuesUndoJob(t *testing.T) {
	s, q := jobsServer(t)
	sess := s.signedIn(t, "anna")
	src := doneJob(t, q, "ben", "p1.jpg", "p2.jpg")
	path := "/api/jobs/" + strconv.FormatInt(src, 10) + "/undo"

	wantJSONStatus(t, "no session", s.do(http.MethodPost, path, "", withXRW), http.StatusUnauthorized)
	wantJSONStatus(t, "no csrf header", s.do(http.MethodPost, path, "", sess), http.StatusForbidden)

	rec := s.do(http.MethodPost, path, "", sess, withXRW)
	wantJSONStatus(t, "undo", rec, http.StatusAccepted)
	var c struct{ ID int64 }
	if err := json.Unmarshal(rec.Body.Bytes(), &c); err != nil || c.ID == 0 {
		t.Fatalf("body %q has no id", rec.Body.String())
	}
	job, items, err := q.Get(context.Background(), c.ID)
	if err != nil {
		t.Fatal(err)
	}
	if job.Action != "undo" || job.UndoOf != src || job.Username != "anna" || job.Status != writer.JobQueued || len(items) != 2 {
		t.Errorf("undo job = %+v with %d items", job, len(items))
	}

	rec = s.do(http.MethodGet, "/api/jobs", "", sess, withXRW)
	var list []struct {
		ID       int64  `json:"id"`
		Action   string `json:"action"`
		UndoOf   *int64 `json:"undo_of"`
		UndoneBy *int64 `json:"undone_by"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil || len(list) != 2 {
		t.Fatalf("list = %s, %v", rec.Body.String(), err)
	}
	if u := list[0]; u.ID != c.ID || u.Action != "undo" || u.UndoOf == nil || *u.UndoOf != src {
		t.Errorf("list[0] = %+v, want the undo job pointing at %d", u, src)
	}
	if o := list[1]; o.UndoneBy == nil || *o.UndoneBy != c.ID {
		t.Errorf("list[1] = %+v, want undone_by %d", o, c.ID)
	}
}

func TestUndoEndpointRejectsActiveOrAlreadyUndoneJob(t *testing.T) {
	s, q := jobsServer(t)
	sess := s.signedIn(t, "anna")
	undo := func(id string) *httptest.ResponseRecorder {
		return s.do(http.MethodPost, "/api/jobs/"+id+"/undo", "", sess, withXRW)
	}

	rec := s.do(http.MethodPost, "/api/jobs",
		`{"selection":{"include_ids":[1]},"action":"tags.add","payload":{"tags":["A"]}}`, sess, withXRW)
	var queued struct{ ID int64 }
	if err := json.Unmarshal(rec.Body.Bytes(), &queued); err != nil {
		t.Fatal(err)
	}
	wantJSONStatus(t, "queued job", undo(strconv.FormatInt(queued.ID, 10)), http.StatusConflict)

	src := doneJob(t, q, "anna", "p1.jpg")
	id := strconv.FormatInt(src, 10)
	wantJSONStatus(t, "first undo", undo(id), http.StatusAccepted)
	rec = undo(id)
	wantJSONStatus(t, "second undo", rec, http.StatusConflict)
	if !strings.Contains(rec.Body.String(), "already undone") {
		t.Errorf("body %s lacks a clear message", rec.Body.String())
	}

	wantJSONStatus(t, "unknown id", undo("999"), http.StatusNotFound)
	wantJSONStatus(t, "non-numeric id", undo("abc"), http.StatusNotFound)
}

func TestJobListShowsUserTimeActionCountAndResult(t *testing.T) {
	s, q := jobsServer(t)
	sess := s.signedIn(t, "anna")
	src := doneJob(t, q, "ben", "p1.jpg", "p2.jpg")
	if _, err := q.DB.Exec(`UPDATE jobs SET ok = 1, skipped = 1 WHERE id = ?`, src); err != nil {
		t.Fatal(err)
	}

	rec := s.do(http.MethodGet, "/api/jobs", "", sess, withXRW)
	wantJSONStatus(t, "list", rec, http.StatusOK)
	var list []struct {
		User      string  `json:"user"`
		CreatedAt *string `json:"created_at"`
		Action    string  `json:"action"`
		Status    string  `json:"status"`
		Total     int     `json:"total"`
		OK        int     `json:"ok"`
		Skipped   int     `json:"skipped"`
		Failed    int     `json:"failed"`
		UndoOf    *int64  `json:"undo_of"`
		UndoneBy  *int64  `json:"undone_by"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil || len(list) != 1 {
		t.Fatalf("list = %s, %v", rec.Body.String(), err)
	}
	j := list[0]
	if j.User != "ben" || j.CreatedAt == nil || j.Action != "tags.add" || j.Status != "done" ||
		j.Total != 2 || j.OK != 1 || j.Skipped != 1 || j.Failed != 0 || j.UndoOf != nil || j.UndoneBy != nil {
		t.Errorf("job = %+v (body %s)", j, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "old") || strings.Contains(rec.Body.String(), "new_values") {
		t.Errorf("journal values leak into the list: %s", rec.Body.String())
	}
}

func tagPaths(t *testing.T, s *testServer, sess reqOpt) []string {
	t.Helper()
	out := []string{}
	var walk func(nodes []tagNode)
	walk = func(nodes []tagNode) {
		for _, n := range nodes {
			out = append(out, n.Path)
			walk(n.Children)
		}
	}
	walk(fetchTags(t, s, sess))
	return out
}

func TestAssigningANewTagCreatesItInTheTagTreeEvenBeforeTheJobRuns(t *testing.T) {
	s, q := jobsServer(t)
	sess := s.signedIn(t, "anna")
	rec := s.do(http.MethodPost, "/api/jobs",
		`{"selection":{"include_ids":[1]},"action":"tags.add","payload":{"tags":[" Anlass / Geburtstag "]}}`, sess, withXRW)
	wantJSONStatus(t, "create", rec, http.StatusAccepted)

	jobs, err := q.List(context.Background(), 10)
	if err != nil || len(jobs) != 1 || jobs[0].Status != writer.JobQueued {
		t.Fatalf("jobs = %+v, %v; want one job that has not run", jobs, err)
	}
	if got := tagPaths(t, s, sess); !reflect.DeepEqual(got, []string{"Anlass", "Anlass/Geburtstag"}) {
		t.Errorf("tree = %v, want the new tag and its parent", got)
	}
}

func TestAnInvalidTagPathIsRejectedWithItsReasonAndCreatesNoJob(t *testing.T) {
	s, q := jobsServer(t)
	sess := s.signedIn(t, "anna")
	for _, action := range []string{"tags.add", "tags.remove"} {
		// The valid first path must not leak into the tree when a later one is bad.
		body := `{"selection":{"include_ids":[1]},"action":"` + action + `","payload":{"tags":["Gut","A//B"]}}`
		rec := s.do(http.MethodPost, "/api/jobs", body, sess, withXRW)
		wantJSONStatus(t, action, rec, http.StatusBadRequest)
		var e map[string]string
		if err := json.Unmarshal(rec.Body.Bytes(), &e); err != nil || e["error"] == "" {
			t.Errorf("%s: body %q lacks a reason", action, rec.Body.String())
		}
	}
	if jobs, err := q.List(context.Background(), 10); err != nil || len(jobs) != 0 {
		t.Errorf("jobs = %v, %v; want none", jobs, err)
	}
	if got := tagPaths(t, s, sess); len(got) != 0 {
		t.Errorf("tree = %v, want unchanged (empty)", got)
	}
}

func TestLocationJobWithABadPayloadIsRefusedAndAValidOneIsQueued(t *testing.T) {
	s, q := jobsServer(t)
	sess := s.signedIn(t, "anna")
	post := func(action, payload string) *httptest.ResponseRecorder {
		return s.do(http.MethodPost, "/api/jobs",
			`{"selection":{"include_ids":[1]},"action":"`+action+`","payload":`+payload+`}`, sess, withXRW)
	}
	for name, c := range map[string][2]string{
		"no location":  {"location.set", `{"mode":"all"}`},
		"missing mode": {"location.set", `{"location":{"lat":1,"lon":2,"label":"X"}}`},
		"unknown mode": {"location.set", `{"mode":"some","location":{"lat":1,"lon":2,"label":"X"}}`},
		"no latitude":  {"location.set", `{"mode":"all","location":{"lon":2,"label":"X"}}`},
		"no longitude": {"location.set", `{"mode":"all","location":{"lat":1,"label":"X"}}`},
		"latitude":     {"location.set", `{"mode":"all","location":{"lat":91,"lon":2,"label":"X"}}`},
		"longitude":    {"location.set", `{"mode":"all","location":{"lat":1,"lon":181,"label":"X"}}`},
		"empty name":   {"location.set", `{"mode":"all","location":{"lat":1,"lon":2,"label":" "}}`},
		"control char": {"location.set", `{"mode":"all","location":{"lat":1,"lon":2,"label":"X","city":"a\u0000b"}}`},
	} {
		rec := post(c[0], c[1])
		wantJSONStatus(t, name, rec, http.StatusBadRequest)
		var e map[string]string
		if err := json.Unmarshal(rec.Body.Bytes(), &e); err != nil || e["error"] == "" {
			t.Errorf("%s: body %q lacks a reason", name, rec.Body.String())
		}
	}
	if jobs, err := q.List(context.Background(), 10); err != nil || len(jobs) != 0 {
		t.Fatalf("jobs = %v, %v; a refused request must queue nothing", jobs, err)
	}

	wantJSONStatus(t, "set", post("location.set",
		`{"mode":"missing","location":{"lat":48.2,"lon":16.37,"label":" Stephansdom ","city":"Wien","countryCode":"at"}}`), http.StatusAccepted)
	wantJSONStatus(t, "remove", post("location.remove", `{}`), http.StatusAccepted)

	jobs, err := q.List(context.Background(), 10)
	if err != nil || len(jobs) != 2 {
		t.Fatalf("jobs = %v, %v; want two", jobs, err)
	}
	var payload string
	if err := q.DB.QueryRow(`SELECT payload FROM jobs WHERE action = 'location.set'`).Scan(&payload); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(payload, `"Label":"Stephansdom"`) || !strings.Contains(payload, `"LocationMode":"missing"`) || !strings.Contains(payload, `"ClearLocation":true`) {
		t.Errorf("payload = %s, want trimmed label, mode and ClearLocation", payload)
	}
}

func TestRemovingATagQueuesARemoveJobWithoutCreatingTheTag(t *testing.T) {
	s, q := jobsServer(t)
	sess := s.signedIn(t, "anna")
	rec := s.do(http.MethodPost, "/api/jobs",
		`{"selection":{"include_ids":[1,2]},"action":"tags.remove","payload":{"tags":["Anlass/Geburtstag"]}}`, sess, withXRW)
	wantJSONStatus(t, "create", rec, http.StatusAccepted)
	var created struct{ ID int64 }
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	job, _, err := q.Get(context.Background(), created.ID)
	if err != nil || job.Action != "tags.remove" || job.Total != 2 {
		t.Fatalf("job = %+v, %v; want tags.remove over 2 files", job, err)
	}
	var payload string
	if err := q.DB.QueryRow(`SELECT payload FROM jobs WHERE id = ?`, created.ID).Scan(&payload); err != nil {
		t.Fatal(err)
	}
	if want := `{"AddTags":null,"RemoveTags":["Anlass/Geburtstag"],"Location":null,"ReviewState":""}`; payload != want {
		t.Errorf("payload = %s, want %s", payload, want)
	}
	if got := tagPaths(t, s, sess); len(got) != 0 {
		t.Errorf("tree = %v, remove must not create tags", got)
	}
}
