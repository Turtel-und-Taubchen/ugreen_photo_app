package api

import (
	"context"
	"encoding/base64"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"photocurator/internal/library"
	"photocurator/internal/thumbs"
)

const (
	defaultPageSize = 200
	maxPageSize     = 500
)

var monthPattern = regexp.MustCompile(`^\d{4}-(0[1-9]|1[0-2])$`)

// PhotoLister reads the index for the grid and the detail view. An empty month lists the
// whole library.
type PhotoLister interface {
	List(ctx context.Context, month string, after *library.ListCursor, limit int) ([]library.Photo, error)
	Groups(ctx context.Context) (library.Groups, error)
	Detail(ctx context.Context, id int64) (library.Detail, bool, error)
}

type photoJSON struct {
	ID        int64   `json:"id"`
	RelPath   string  `json:"relPath"`
	MediaType string  `json:"mediaType"`
	TakenAt   *string `json:"takenAt"`
	ThumbKey  string  `json:"thumbKey"`
}

type dayJSON struct {
	Day   string `json:"day"`
	Count int    `json:"count"`
}

type monthJSON struct {
	Month string    `json:"month"`
	Count int       `json:"count"`
	Days  []dayJSON `json:"days"`
}

type folderJSON struct {
	Folder string `json:"folder"`
	Count  int    `json:"count"`
}

type placeJSON struct {
	Label       string   `json:"label"`
	Sublocation string   `json:"sublocation"`
	City        string   `json:"city"`
	State       string   `json:"state"`
	Country     string   `json:"country"`
	Lat         *float64 `json:"lat"`
	Lon         *float64 `json:"lon"`
}

type detailJSON struct {
	ID          int64     `json:"id"`
	RelPath     string    `json:"rel_path"`
	Folder      string    `json:"folder"`
	MediaType   string    `json:"media_type"`
	Format      string    `json:"format"`
	TakenAt     *string   `json:"taken_at"`
	TakenSource string    `json:"taken_source"`
	ThumbKey    string    `json:"thumb_key"`
	Tags        []string  `json:"tags"`
	Location    placeJSON `json:"location"`
	ReviewState string    `json:"review_state"`
	ReadError   *string   `json:"read_error"`
	PrevID      *int64    `json:"prev_id"`
	NextID      *int64    `json:"next_id"`
}

func badRequest(w http.ResponseWriter, msg string) {
	writeJSON(w, http.StatusBadRequest, map[string]string{"error": msg})
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
				badRequest(w, "invalid limit")
				return
			}
			limit = min(n, maxPageSize)
		}
		month := q.Get("month")
		if month != "" && !monthPattern.MatchString(month) {
			badRequest(w, "invalid month")
			return
		}
		var after *library.ListCursor
		if v := q.Get("after"); v != "" {
			c, ok := decodeCursor(v)
			if !ok {
				badRequest(w, "invalid cursor")
				return
			}
			after = c
		}
		// One extra row tells whether another page exists without a second query.
		rows, err := l.List(r.Context(), month, after, limit+1)
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

	mux.HandleFunc("GET /api/photos/groups", func(w http.ResponseWriter, r *http.Request) {
		g, err := l.Groups(r.Context())
		if err != nil {
			slog.Error("photo groups", "err", err)
			internalError(w)
			return
		}
		months := make([]monthJSON, 0, len(g.Months))
		for _, m := range g.Months {
			days := make([]dayJSON, 0, len(m.Days))
			for _, d := range m.Days {
				days = append(days, dayJSON{d.Day, d.Count})
			}
			months = append(months, monthJSON{m.Month, m.Count, days})
		}
		folders := make([]folderJSON, 0, len(g.Folders))
		for _, f := range g.Folders {
			folders = append(folders, folderJSON{f.Folder, f.Count})
		}
		writeJSON(w, http.StatusOK, map[string]any{"total": g.Total, "months": months, "folders": folders})
	})

	mux.HandleFunc("GET /api/photos/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			notFound(w)
			return
		}
		d, ok, err := l.Detail(r.Context(), id)
		if err != nil {
			slog.Error("photo detail", "err", err)
			internalError(w)
			return
		}
		if !ok {
			notFound(w)
			return
		}
		writeJSON(w, http.StatusOK, detailJSON{
			ID: d.ID, RelPath: d.RelPath, Folder: d.Folder, MediaType: d.MediaType, Format: d.Format,
			TakenAt: d.TakenAt, TakenSource: d.TakenSource, ThumbKey: thumbs.Key(d.RelPath, d.Size, d.MtimeNs),
			Tags: d.Tags, Location: placeJSON(d.Place), ReviewState: d.ReviewState, ReadError: d.ReadError,
			PrevID: d.PrevID, NextID: d.NextID,
		})
	})
}
