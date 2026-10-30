package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"sort"
	"strconv"
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
	return newTestServerDeps(t, api.LibraryDeps{Index: selectionIndex(t), Writes: q}), q
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
		"unknown action":  `{` + sel + `,"action":"tags.remove","payload":{"tags":["A"]}}`,
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
