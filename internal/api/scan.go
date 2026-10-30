package api

import (
	"database/sql"
	"log/slog"
	"net/http"
	"time"

	"photocurator/internal/library"
)

// unreadableLimit caps the list in the response; unreadable_count carries the real total.
const unreadableLimit = 50

// ScanControl is what the scan endpoints need from the library.
type ScanControl interface {
	Status() library.Status
	Trigger() bool
	Count() (int, error)
	Unreadable(limit int) ([]library.UnreadableFile, int, error)
}

// LibraryDeps are the library services behind the API; a nil field leaves its routes unregistered.
type LibraryDeps struct {
	Scan   ScanControl
	Thumbs *ThumbDeps
	Photos PhotoLister
	// Index is index.db; it serves the selection endpoints.
	Index *sql.DB
	// Tags serves /api/tags; nil leaves the routes unregistered.
	Tags *TagDeps
}

type unreadableJSON struct {
	RelPath string `json:"rel_path"`
	Error   string `json:"error"`
}

type scanJSON struct {
	State           string           `json:"state"`
	Trigger         string           `json:"trigger"`
	Done            int              `json:"done"`
	Total           int              `json:"total"`
	StartedAt       *string          `json:"started_at"`
	FinishedAt      *string          `json:"finished_at"`
	LastError       string           `json:"last_error"`
	Files           int              `json:"files"`
	Unreadable      []unreadableJSON `json:"unreadable"`
	UnreadableCount int              `json:"unreadable_count"`
}

// rfc3339 maps the zero time ("never happened") to JSON null.
func rfc3339(t time.Time) *string {
	if t.IsZero() {
		return nil
	}
	s := t.UTC().Format(time.RFC3339)
	return &s
}

func scanReport(c ScanControl) (scanJSON, error) {
	st := c.Status()
	files, err := c.Count()
	if err != nil {
		return scanJSON{}, err
	}
	bad, total, err := c.Unreadable(unreadableLimit)
	if err != nil {
		return scanJSON{}, err
	}
	list := make([]unreadableJSON, 0, len(bad)) // never nil: the UI expects an array
	for _, u := range bad {
		list = append(list, unreadableJSON{RelPath: u.RelPath, Error: u.Error})
	}
	return scanJSON{
		State: st.State, Trigger: st.Trigger, Done: st.Done, Total: st.Total,
		StartedAt: rfc3339(st.StartedAt), FinishedAt: rfc3339(st.FinishedAt), LastError: st.LastError,
		Files: files, Unreadable: list, UnreadableCount: total,
	}, nil
}

func scanHandler(c ScanControl, status int, trigger bool) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		if trigger {
			// A refused trigger (already running or queued) is not an error: the report shows the running scan.
			c.Trigger()
		}
		rep, err := scanReport(c)
		if err != nil {
			slog.Error("scan status", "err", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}
		writeJSON(w, status, rep)
	}
}

func registerScan(mux *http.ServeMux, c ScanControl) {
	mux.HandleFunc("GET /api/scan", scanHandler(c, http.StatusOK, false))
	mux.HandleFunc("POST /api/scan", scanHandler(c, http.StatusAccepted, true))
}
