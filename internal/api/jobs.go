package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"photocurator/internal/library"
	"photocurator/internal/metadata"
	"photocurator/internal/tags"
	"photocurator/internal/writer"
)

const (
	actionTagsAdd = "tags.add"
	jobListLimit  = 50
)

type jobRequest struct {
	Selection library.Selection `json:"selection"`
	Action    string            `json:"action"`
	Payload   struct {
		Tags []string `json:"tags"`
	} `json:"payload"`
}

type jobJSON struct {
	ID         int64   `json:"id"`
	User       string  `json:"user"`
	Action     string  `json:"action"`
	Status     string  `json:"status"`
	Total      int     `json:"total"`
	Done       int     `json:"done"`
	OK         int     `json:"ok"`
	Skipped    int     `json:"skipped"`
	Failed     int     `json:"failed"`
	CreatedAt  *string `json:"created_at"`
	FinishedAt *string `json:"finished_at"`
	UndoOf     *int64  `json:"undo_of"`
	UndoneBy   *int64  `json:"undone_by"`
}

type jobItemJSON struct {
	RelPath string `json:"rel_path"`
	State   string `json:"state"`
	Reason  string `json:"reason"`
}

type jobDetailJSON struct {
	jobJSON
	Items []jobItemJSON `json:"items"`
}

// idOrNil maps the store's 0 for "no such job" to JSON null.
func idOrNil(id int64) *int64 {
	if id == 0 {
		return nil
	}
	return &id
}

func toJobJSON(j writer.Job) jobJSON {
	return jobJSON{
		ID: j.ID, User: j.Username, UndoOf: idOrNil(j.UndoOf), UndoneBy: idOrNil(j.UndoneBy), Action: j.Action, Status: j.Status, Total: j.Total, Done: j.Done,
		OK: j.OK, Skipped: j.Skipped, Failed: j.Failed,
		CreatedAt: rfc3339(j.CreatedAt), FinishedAt: rfc3339(j.FinishedAt),
	}
}

func logInternal(w http.ResponseWriter, what string, err error) {
	slog.Error(what, "err", err)
	internalError(w)
}

// registerJobs serves the write-job endpoints; idx resolves a selection to files.
func registerJobs(mux *http.ServeMux, q *writer.Queue, idx *sql.DB) {
	mux.HandleFunc("POST /api/jobs", func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxSelectionBody)
		var req jobRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			badRequest(w, "invalid job body")
			return
		}
		if req.Action != actionTagsAdd {
			badRequest(w, "unknown action")
			return
		}
		if len(req.Payload.Tags) == 0 {
			badRequest(w, "payload.tags must not be empty")
			return
		}
		// The canonical path is what the metadata mapping splits into terms and hierarchy.
		change := metadata.Change{}
		for _, t := range req.Payload.Tags {
			p, err := tags.ParsePath(t)
			if err != nil {
				badRequest(w, err.Error())
				return
			}
			change.AddTags = append(change.AddTags, p)
		}
		if err := req.Selection.Validate(); err != nil {
			badRequest(w, err.Error())
			return
		}
		files, err := library.ResolveSelection(r.Context(), idx, req.Selection)
		if errors.Is(err, library.ErrInvalidSelection) {
			badRequest(w, err.Error())
			return
		}
		if err != nil {
			logInternal(w, "resolve selection", err)
			return
		}
		if len(files) == 0 {
			badRequest(w, "selection is empty")
			return
		}
		rels := make([]string, len(files))
		for i, f := range files {
			rels[i] = f.RelPath
		}
		user, _ := UserFromContext(r.Context())
		id, err := q.Create(r.Context(), user, req.Action, change, rels)
		if err != nil {
			logInternal(w, "create job", err)
			return
		}
		writeJSON(w, http.StatusAccepted, map[string]int64{"id": id})
	})

	mux.HandleFunc("GET /api/jobs", func(w http.ResponseWriter, r *http.Request) {
		jobs, err := q.List(r.Context(), jobListLimit)
		if err != nil {
			logInternal(w, "list jobs", err)
			return
		}
		out := make([]jobJSON, 0, len(jobs)) // never nil: the UI expects an array
		for _, j := range jobs {
			out = append(out, toJobJSON(j))
		}
		writeJSON(w, http.StatusOK, out)
	})

	mux.HandleFunc("POST /api/jobs/{id}/undo", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		user, _ := UserFromContext(r.Context())
		undoID, err := q.Undo(r.Context(), id, user)
		switch {
		case errors.Is(err, sql.ErrNoRows):
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		case errors.Is(err, writer.ErrAlreadyUndone):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "job was already undone"})
		case errors.Is(err, writer.ErrJobActive):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "job is still queued or running"})
		case err != nil:
			logInternal(w, "undo job", err)
		default:
			writeJSON(w, http.StatusAccepted, map[string]int64{"id": undoID})
		}
	})

	mux.HandleFunc("GET /api/jobs/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		j, items, err := q.Get(r.Context(), id)
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		if err != nil {
			logInternal(w, "get job", err)
			return
		}
		d := jobDetailJSON{jobJSON: toJobJSON(j), Items: make([]jobItemJSON, 0, len(items))}
		for _, it := range items {
			d.Items = append(d.Items, jobItemJSON{RelPath: it.RelPath, State: it.State, Reason: it.Reason})
		}
		writeJSON(w, http.StatusOK, d)
	})
}
