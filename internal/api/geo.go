package api

import (
	"context"
	"errors"
	"math"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"photocurator/internal/geo"
)

const (
	maxQueryRunes = 200

	geoDisabledMsg = "Ortssuche deaktiviert: Es ist kein Nominatim-Kontakt konfiguriert (NOMINATIM_CONTACT)."
	geoUpstreamMsg = "Ortssuche gerade nicht erreichbar. Du kannst den Ort per Koordinaten und Namen selbst setzen."
)

// Geocoder is what the place endpoints need from internal/geo.
type Geocoder interface {
	Enabled() bool
	Search(ctx context.Context, q string) ([]geo.Place, error)
	Reverse(ctx context.Context, lat, lon float64) (*geo.Place, error)
}

func registerGeo(mux *http.ServeMux, g Geocoder) {
	mux.HandleFunc("GET /api/geo/status", func(w http.ResponseWriter, _ *http.Request) {
		if g.Enabled() {
			writeJSON(w, http.StatusOK, map[string]any{"enabled": true})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"enabled": false, "reason": geoDisabledMsg})
	})
	mux.HandleFunc("GET /api/geo/search", func(w http.ResponseWriter, r *http.Request) {
		q := strings.TrimSpace(r.URL.Query().Get("q"))
		if q == "" || utf8.RuneCountInString(q) > maxQueryRunes {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Bitte einen Suchbegriff mit höchstens 200 Zeichen eingeben."})
			return
		}
		places, err := g.Search(r.Context(), q)
		if err != nil {
			writeGeoError(w, err)
			return
		}
		if places == nil {
			places = []geo.Place{}
		}
		writeJSON(w, http.StatusOK, map[string]any{"results": places})
	})
	mux.HandleFunc("GET /api/geo/reverse", func(w http.ResponseWriter, r *http.Request) {
		lat, okLat := parseCoord(r.URL.Query().Get("lat"), 90)
		lon, okLon := parseCoord(r.URL.Query().Get("lon"), 180)
		if !okLat || !okLon {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Ungültige Koordinaten: Breite -90 bis 90, Länge -180 bis 180."})
			return
		}
		place, err := g.Reverse(r.Context(), lat, lon)
		if err != nil {
			writeGeoError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"place": place})
	})
}

// parseCoord rejects NaN and Inf, which ParseFloat accepts and range checks alone would not catch.
func parseCoord(s string, limit float64) (float64, bool) {
	v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil || math.IsNaN(v) || math.IsInf(v, 0) || v < -limit || v > limit {
		return 0, false
	}
	return v, true
}

func writeGeoError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, geo.ErrDisabled):
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": geoDisabledMsg})
	case errors.Is(err, geo.ErrUpstream):
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": geoUpstreamMsg})
	default:
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Interner Fehler."})
	}
}
