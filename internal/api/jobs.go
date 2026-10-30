package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"photocurator/internal/geo"
	"photocurator/internal/library"
	"photocurator/internal/metadata"
	"photocurator/internal/tags"
	"photocurator/internal/writer"
)

const (
	actionTagsAdd    = "tags.add"
	actionTagsRemove = "tags.remove"

	actionLocationSet    = "location.set"
	actionLocationRemove = "location.remove"
	jobListLimit         = 50
)

type jobRequest struct {
	Selection library.Selection `json:"selection"`
	Action    string            `json:"action"`
	Payload   struct {
		Tags     []string       `json:"tags"`
		Location *locationInput `json:"location"`
		Mode     string         `json:"mode"`
	} `json:"payload"`
}

type locationInput struct {
	Lat         *float64 `json:"lat"` // pointers: an absent coordinate must not become (0, 0)
	Lon         *float64 `json:"lon"`
	Label       string   `json:"label"`
	Sublocation string   `json:"sublocation"`
	City        string   `json:"city"`
	State       string   `json:"state"`
	Country     string   `json:"country"`
	CountryCode string   `json:"countryCode"`
}

// locationSetChange validates the payload and builds the change. ClearLocation rides along so
// that parts the new place leaves empty do not keep an older file's value.
func locationSetChange(in *locationInput, mode string) (metadata.Change, error) {
	if in == nil {
		return metadata.Change{}, errors.New("payload.location fehlt")
	}
	if mode != metadata.LocationModeAll && mode != metadata.LocationModeMissing {
		return metadata.Change{}, errors.New("payload.mode muss \"all\" oder \"missing\" sein")
	}
	if in.Lat == nil || in.Lon == nil {
		return metadata.Change{}, errors.New("payload.location.lat und lon sind erforderlich")
	}
	if err := geo.ValidatePoint(*in.Lat, *in.Lon); err != nil {
		return metadata.Change{}, err
	}
	label, parts, err := geo.ValidateLocationText(in.Label, geo.Parts{
		Sublocation: in.Sublocation, City: in.City, State: in.State, Country: in.Country, CountryCode: in.CountryCode,
	})
	if err != nil {
		return metadata.Change{}, err
	}
	// countryCode is validated and normalised above but not written: the mapping has no field for it.
	return metadata.Change{
		Location: &metadata.Location{
			Lat: *in.Lat, Lon: *in.Lon, Label: label,
			Sublocation: parts.Sublocation, City: parts.City, State: parts.State, Country: parts.Country,
		},
		ClearLocation: true,
		LocationMode:  mode,
	}, nil
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
func registerJobs(mux *http.ServeMux, q *writer.Queue, idx *sql.DB, td *TagDeps) {
	mux.HandleFunc("POST /api/jobs", func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxSelectionBody)
		var req jobRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			badRequest(w, "invalid job body")
			return
		}
		var paths []string
		var change metadata.Change
		switch req.Action {
		case actionTagsAdd, actionTagsRemove:
			if len(req.Payload.Tags) == 0 {
				badRequest(w, "payload.tags must not be empty")
				return
			}
			// The canonical path is what the metadata mapping splits into terms and hierarchy.
			paths = make([]string, 0, len(req.Payload.Tags))
			for _, t := range req.Payload.Tags {
				p, err := tags.ParsePath(t)
				if err != nil {
					badRequest(w, err.Error())
					return
				}
				paths = append(paths, p)
			}
			if req.Action == actionTagsRemove {
				change.RemoveTags = paths
			} else {
				change.AddTags = paths
			}
		case actionLocationSet:
			c, err := locationSetChange(req.Payload.Location, req.Payload.Mode)
			if err != nil {
				badRequest(w, err.Error())
				return
			}
			change = c
		case actionLocationRemove:
			change.ClearLocation = true
		default:
			badRequest(w, "unknown action")
			return
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
		// Created only once the request is known to be good and before enqueuing: the tag must
		// exist in the tree even if every single file later fails. Create is idempotent.
		if req.Action == actionTagsAdd {
			for _, p := range paths {
				if _, err := tags.Create(r.Context(), td.App, p); err != nil {
					logInternal(w, "create tag", err)
					return
				}
			}
		}
		user, _ := UserFromContext(r.Context())
		id, err := q.Create(r.Context(), user, req.Action, change, rels)
		if err != nil {
			logInternal(w, "create job", err)
			return
		}
		// The job is queued and will run whatever happens here, so a failure only costs the
		// recent-tags hint and is not reported to the client.
		if req.Action == actionTagsAdd {
			if err := tags.TouchRecent(r.Context(), td.App, paths); err != nil {
				slog.Error("touch recent tags", "err", err)
			}
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
