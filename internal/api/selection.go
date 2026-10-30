package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"photocurator/internal/library"
)

const maxSelectionBody = 1 << 20

func registerSelection(mux *http.ServeMux, idx *sql.DB) {
	mux.HandleFunc("POST /api/selection/count", func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxSelectionBody)
		var sel library.Selection
		if err := json.NewDecoder(r.Body).Decode(&sel); err != nil {
			badRequest(w, "invalid selection body")
			return
		}
		if err := sel.Validate(); err != nil {
			badRequest(w, err.Error())
			return
		}
		n, located, err := library.CountSelectionLocated(r.Context(), idx, sel)
		if errors.Is(err, library.ErrInvalidSelection) {
			badRequest(w, err.Error())
			return
		}
		if err != nil {
			slog.Error("count selection", "err", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]int{"count": n, "located": located})
	})

	mux.HandleFunc("POST /api/selection/tags", func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxSelectionBody)
		var sel library.Selection
		if err := json.NewDecoder(r.Body).Decode(&sel); err != nil {
			badRequest(w, "invalid selection body")
			return
		}
		if err := sel.Validate(); err != nil {
			badRequest(w, err.Error())
			return
		}
		counts, err := library.SelectionTagCounts(r.Context(), idx, sel)
		if errors.Is(err, library.ErrInvalidSelection) {
			badRequest(w, err.Error())
			return
		}
		if err != nil {
			slog.Error("selection tags", "err", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}
		writeJSON(w, http.StatusOK, counts)
	})
}
