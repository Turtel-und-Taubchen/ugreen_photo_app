package api

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"path"
	"strings"

	"photocurator/internal/version"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// NewHandler serves the JSON API under /api/ and the SPA from static.
func NewHandler(static fs.FS) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "version": version.Version})
	})
	// Unknown API paths must never fall through to the SPA's index.html.
	mux.HandleFunc("/api/", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
	})
	mux.Handle("/", spa(static))
	return mux
}

func spa(static fs.FS) http.Handler {
	files := http.FileServerFS(static)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
		if name != "" {
			if info, err := fs.Stat(static, name); err != nil || info.IsDir() {
				r.URL.Path = "/"
			}
		}
		files.ServeHTTP(w, r)
	})
}
