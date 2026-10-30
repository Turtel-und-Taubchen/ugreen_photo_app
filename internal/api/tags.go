package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"photocurator/internal/library"
	"photocurator/internal/tags"
)

// TagDeps are the two databases behind the tag tree: App holds the user-defined tags, Index
// the tags read from the files.
type TagDeps struct {
	App   *sql.DB
	Index *sql.DB
}

const maxTagBody = 1 << 16

func registerTags(mux *http.ServeMux, d *TagDeps) {
	mux.HandleFunc("GET /api/tags", func(w http.ResponseWriter, r *http.Request) {
		defs, err := tags.List(r.Context(), d.App)
		if err != nil {
			tagsFailed(w, "list tags", err)
			return
		}
		pairs, pairsErr := library.TagPairs(r.Context(), d.Index)
		tree := tags.BuildTree(defs, pairs)
		if err := pairsErr(); err != nil {
			tagsFailed(w, "read tag pairs", err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"tags": tree})
	})

	mux.HandleFunc("GET /api/tags/recent", func(w http.ResponseWriter, r *http.Request) {
		recent, err := tags.ListRecent(r.Context(), d.App)
		if err != nil {
			tagsFailed(w, "list recent tags", err)
			return
		}
		if recent == nil {
			recent = []string{} // never null: the UI expects an array
		}
		writeJSON(w, http.StatusOK, map[string]any{"tags": recent})
	})

	mux.HandleFunc("POST /api/tags", func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxTagBody)
		var body struct {
			Path string `json:"path"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			badRequest(w, "Ungültige Anfrage.")
			return
		}
		path, err := tags.ParsePath(body.Path)
		if err != nil {
			badRequest(w, err.Error())
			return
		}
		created, err := tags.Create(r.Context(), d.App, path)
		if err != nil {
			tagsFailed(w, "create tag", err)
			return
		}
		status := http.StatusOK
		if created {
			status = http.StatusCreated
		}
		writeJSON(w, status, map[string]string{"path": path})
	})

	mux.HandleFunc("DELETE /api/tags", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Query().Get("path")
		if strings.Trim(path, "/") == "" {
			badRequest(w, "Kein Tag angegeben.")
			return
		}
		n, err := library.TagSubtreeCount(r.Context(), d.Index, path)
		if err != nil {
			tagsFailed(w, "count tagged photos", err)
			return
		}
		if n > 0 {
			writeJSON(w, http.StatusConflict, map[string]any{
				"error": fmt.Sprintf("Dieser Tag ist noch %d Fotos zugeordnet. Entferne ihn zuerst über „Tags entfernen“ von den Fotos.", n),
				"count": n,
			})
			return
		}
		// Without photos below it, an index-origin node has nothing left to show, so only a
		// definition can make the path exist.
		defined, err := isDefined(r.Context(), d.App, path)
		if err != nil {
			tagsFailed(w, "look up tag", err)
			return
		}
		if !defined {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "Tag nicht gefunden."})
			return
		}
		if _, err := tags.DeleteSubtree(r.Context(), d.App, path); err != nil {
			tagsFailed(w, "delete tag", err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}

func isDefined(ctx context.Context, db *sql.DB, path string) (bool, error) {
	var one int
	err := db.QueryRowContext(ctx, `SELECT 1 FROM tag_definitions WHERE path = ?`, path).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

func tagsFailed(w http.ResponseWriter, what string, err error) {
	slog.Error(what, "err", err)
	writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
}
