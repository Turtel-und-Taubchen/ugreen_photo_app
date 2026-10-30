package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"photocurator/internal/geo"
)

// registerPlaces serves the favourite places: one list shared by all signed-in users.
func registerPlaces(mux *http.ServeMux, db *sql.DB) {
	mux.HandleFunc("GET /api/places", func(w http.ResponseWriter, r *http.Request) {
		list, err := geo.ListFavorites(r.Context(), db)
		if err != nil {
			tagsFailed(w, "list favorite places", err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"places": list})
	})

	mux.HandleFunc("POST /api/places", func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxTagBody)
		var body geo.Favorite
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			badRequest(w, "Ungültige Anfrage.")
			return
		}
		f, err := geo.CreateFavorite(r.Context(), db, body, time.Now())
		if err != nil {
			placeFailed(w, "create favorite place", err, body.Name)
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"place": f})
	})

	mux.HandleFunc("PATCH /api/places/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, ok := placeID(w, r)
		if !ok {
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxTagBody)
		var body struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			badRequest(w, "Ungültige Anfrage.")
			return
		}
		f, err := geo.RenameFavorite(r.Context(), db, id, body.Name, time.Now())
		if err != nil {
			placeFailed(w, "rename favorite place", err, body.Name)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"place": f})
	})

	mux.HandleFunc("DELETE /api/places/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, ok := placeID(w, r)
		if !ok {
			return
		}
		if err := geo.DeleteFavorite(r.Context(), db, id); err != nil {
			placeFailed(w, "delete favorite place", err, "")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}

func placeID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		badRequest(w, "Ungültige Anfrage.")
		return 0, false
	}
	return id, true
}

// placeFailed maps the store's typed errors to their answers; anything else is a 500.
func placeFailed(w http.ResponseWriter, what string, err error, name string) {
	var in *geo.LocationInputError
	switch {
	case errors.As(err, &in):
		badRequest(w, in.Msg)
	case errors.Is(err, geo.ErrFavoriteNotFound):
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Lieblingsort nicht gefunden."})
	case errors.Is(err, geo.ErrFavoriteExists):
		writeJSON(w, http.StatusConflict, map[string]string{
			"error": fmt.Sprintf("Es gibt schon einen Lieblingsort „%s“.", name),
		})
	default:
		tagsFailed(w, what, err)
	}
}
