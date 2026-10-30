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

// NewHandler serves the JSON API under /api/ and the SPA from static. Every route sits behind
// RequireSession (deny by default) and CSRF; the open exceptions are listed in isOpen.
func NewHandler(static fs.FS, deps AuthDeps, lib LibraryDeps) http.Handler {
	a := newAuthAPI(deps)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "version": version.Version})
	})
	mux.HandleFunc("POST /api/login", a.login)
	mux.HandleFunc("POST /api/logout", a.logout)
	mux.HandleFunc("GET /api/me", a.me)
	if lib.Scan != nil {
		registerScan(mux, lib.Scan)
	}
	if lib.Thumbs != nil {
		registerThumbs(mux, lib.Thumbs)
	}
	if lib.Photos != nil {
		registerPhotos(mux, lib.Photos)
	}
	if lib.Index != nil {
		registerSelection(mux, lib.Index)
	}
	if lib.Tags != nil {
		registerTags(mux, lib.Tags)
	}
	if lib.Geo != nil {
		registerGeo(mux, lib.Geo)
	}
	// Unknown API paths must never fall through to the SPA's index.html.
	mux.HandleFunc("/api/", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
	})
	// The SPA fallback also serves index.html for /login, the one page open without a session.
	mux.Handle("/", spa(static))
	return a.RequireSession(static, CSRF(mux))
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
