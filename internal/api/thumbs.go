package api

import (
	"log/slog"
	"net/http"
	"os"
	"strconv"

	"photocurator/internal/thumbs"
)

// ThumbDeps are what the thumbnail endpoint needs: the index lookup, the render service and
// the cache directory the service writes to.
type ThumbDeps struct {
	Lookup  func(id int64) (thumbs.Entry, bool, error)
	Service interface {
		Status(e thumbs.Entry, size string) thumbs.State
	}
	CacheDir string
}

func notFound(w http.ResponseWriter) {
	writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
}

func internalError(w http.ResponseWriter) {
	writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
}

func registerThumbs(mux *http.ServeMux, d *ThumbDeps) {
	mux.HandleFunc("GET /api/thumbs/{id}/{size}", func(w http.ResponseWriter, r *http.Request) {
		size := r.PathValue("size")
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil || (size != "s" && size != "l") {
			notFound(w)
			return
		}
		e, ok, err := d.Lookup(id)
		if err != nil {
			slog.Error("thumbnail lookup", "err", err)
			internalError(w)
			return
		}
		if !ok {
			notFound(w)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		key := thumbs.Key(e.RelPath, e.Size, e.MtimeNs)
		switch d.Service.Status(e, size) {
		case thumbs.Ready:
			f, err := os.Open(thumbs.Path(d.CacheDir, key, size))
			if err != nil {
				slog.Error("thumbnail open", "err", err)
				internalError(w)
				return
			}
			defer f.Close()
			st, err := f.Stat()
			if err != nil {
				internalError(w)
				return
			}
			// Only a URL carrying the current key can never change its content; a bare URL
			// must be revalidated because the file behind the id may be replaced.
			if r.URL.Query().Get("k") == key {
				w.Header().Set("Cache-Control", "private, max-age=31536000, immutable")
			}
			w.Header().Set("Content-Type", "image/webp")
			http.ServeContent(w, r, "", st.ModTime(), f)
		case thumbs.Pending:
			w.Header().Set("Retry-After", "1")
			writeJSON(w, http.StatusAccepted, map[string]string{"state": "pending"})
		default:
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"state": "failed"})
		}
	})
}
