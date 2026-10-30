package api

import (
	"encoding/base64"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"

	"photocurator/internal/library"
	"photocurator/internal/thumbs"
)

const (
	defaultPageSize = 100
	maxPageSize     = 200
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

// PhotoLister pages through the index, newest first.
type PhotoLister interface {
	List(after *library.ListCursor, limit int) ([]library.Photo, error)
}

type photoJSON struct {
	ID        int64   `json:"id"`
	RelPath   string  `json:"relPath"`
	MediaType string  `json:"mediaType"`
	TakenAt   *string `json:"takenAt"`
	ThumbKey  string  `json:"thumbKey"`
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

func encodeCursor(p library.Photo) string {
	at := ""
	if p.TakenAt != nil {
		at = *p.TakenAt
	}
	return base64.RawURLEncoding.EncodeToString([]byte(at + "|" + strconv.FormatInt(p.ID, 10)))
}

func decodeCursor(s string) (*library.ListCursor, bool) {
	raw, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return nil, false
	}
	at, idStr, ok := strings.Cut(string(raw), "|")
	if !ok {
		return nil, false
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return nil, false
	}
	return &library.ListCursor{TakenAt: at, ID: id}, true
}

func registerPhotos(mux *http.ServeMux, l PhotoLister) {
	mux.HandleFunc("GET /api/photos", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		limit := defaultPageSize
		if v := q.Get("limit"); v != "" {
			n, err := strconv.Atoi(v)
			if err != nil || n < 1 {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid limit"})
				return
			}
			limit = min(n, maxPageSize)
		}
		var after *library.ListCursor
		if v := q.Get("after"); v != "" {
			c, ok := decodeCursor(v)
			if !ok {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid cursor"})
				return
			}
			after = c
		}
		// One extra row tells whether another page exists without a second query.
		rows, err := l.List(after, limit+1)
		if err != nil {
			slog.Error("photo list", "err", err)
			internalError(w)
			return
		}
		var next *string
		if len(rows) > limit {
			rows = rows[:limit]
			c := encodeCursor(rows[limit-1])
			next = &c
		}
		items := make([]photoJSON, 0, len(rows))
		for _, p := range rows {
			items = append(items, photoJSON{ID: p.ID, RelPath: p.RelPath, MediaType: p.MediaType, TakenAt: p.TakenAt,
				ThumbKey: thumbs.Key(p.RelPath, p.Size, p.MtimeNs)})
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items, "next": next})
	})
}
