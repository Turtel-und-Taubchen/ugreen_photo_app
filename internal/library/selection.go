package library

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// ErrInvalidSelection marks a selection the caller got wrong (bad key, unknown id, too big),
// as opposed to a database failure; check with errors.Is.
var ErrInvalidSelection = errors.New("invalid selection")

const (
	maxSelectionIDs     = 10000
	maxSelectionFilters = 500
)

// SelectionFilter describes files by group and by span. Ranges are [fromID, toID] pairs.
type SelectionFilter struct {
	Groups []string   `json:"groups"` // "2024-07" or "2024-07-15"
	Ranges [][2]int64 `json:"ranges"`
}

// Selection is a set of files expressed without listing them all: everything matched by the
// filter or by include_ids, except exclude_ids (which always wins).
type Selection struct {
	Filter     SelectionFilter `json:"filter"`
	IncludeIDs []int64         `json:"include_ids"`
	ExcludeIDs []int64         `json:"exclude_ids"`
}

// Resolved is one selected file.
type Resolved struct {
	ID      int64
	RelPath string
}

func invalidf(format string, a ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalidSelection, fmt.Sprintf(format, a...))
}

// Validate checks everything that needs no database: key formats and list sizes.
func (s Selection) Validate() error {
	if len(s.IncludeIDs) > maxSelectionIDs || len(s.ExcludeIDs) > maxSelectionIDs {
		return invalidf("more than %d ids in a list", maxSelectionIDs)
	}
	if len(s.Filter.Groups)+len(s.Filter.Ranges) > maxSelectionFilters {
		return invalidf("more than %d groups and ranges", maxSelectionFilters)
	}
	for _, g := range s.Filter.Groups {
		if _, err := isMonthKey(g); err != nil {
			return err
		}
	}
	return nil
}

// isMonthKey tells a month key from a day key. The length check pins the zero-padded form
// that the SQL key expressions produce; time.Parse then rejects impossible dates.
func isMonthKey(key string) (month bool, err error) {
	switch len(key) {
	case 7:
		if _, err := time.Parse("2006-01", key); err == nil {
			return true, nil
		}
	case 10:
		if _, err := time.Parse("2006-01-02", key); err == nil {
			return false, nil
		}
	}
	return false, invalidf("group key %q is neither JJJJ-MM nor JJJJ-MM-TT", key)
}

type rangeEnd struct {
	At string // COALESCE(taken_at, ''): the grid's ordering key
	ID int64
}

// selectionWhere builds one parameterised predicate for resolve and count, so both always
// agree. ok is false when the selection can match nothing; an empty filter never means "all".
// Lists travel as JSON arrays read through json_each, never spliced into the SQL.
func selectionWhere(ctx context.Context, db *sql.DB, s Selection) (where string, args []any, ok bool, err error) {
	if err := s.Validate(); err != nil {
		return "", nil, false, err
	}
	months, days := []string{}, []string{}
	for _, g := range s.Filter.Groups {
		if m, _ := isMonthKey(g); m {
			months = append(months, g)
		} else {
			days = append(days, g)
		}
	}
	ranges, err := resolveRanges(ctx, db, s.Filter.Ranges)
	if err != nil {
		return "", nil, false, err
	}
	if len(months)+len(days)+len(ranges)+len(s.IncludeIDs) == 0 {
		return "", nil, false, nil
	}
	enc := func(v any) string { b, _ := json.Marshal(v); return string(b) }
	idList := func(v []int64) string {
		if v == nil {
			v = []int64{}
		}
		return enc(v)
	}
	type span struct {
		LoAt string `json:"lo_at"`
		LoID int64  `json:"lo_id"`
		HiAt string `json:"hi_at"`
		HiID int64  `json:"hi_id"`
	}
	spans := []span{}
	for _, r := range ranges {
		spans = append(spans, span{r[0].At, r[0].ID, r[1].At, r[1].ID})
	}
	// Inside the json_each subquery a bare id would be json_each's own column, hence photos.id.
	// Files without a taken date have no group key, so no group matches them. In ranges they
	// follow the grid and sort last (key ''), so they may be endpoints and lie between them.
	where = `(` + MonthKeySQL + ` IN (SELECT value FROM json_each(?))
		OR ` + DayKeySQL + ` IN (SELECT value FROM json_each(?))
		OR id IN (SELECT value FROM json_each(?))
		OR EXISTS (SELECT 1 FROM json_each(?) r
			WHERE (COALESCE(taken_at, '') > json_extract(r.value, '$.lo_at')
			       OR (COALESCE(taken_at, '') = json_extract(r.value, '$.lo_at') AND photos.id >= json_extract(r.value, '$.lo_id')))
			  AND (COALESCE(taken_at, '') < json_extract(r.value, '$.hi_at')
			       OR (COALESCE(taken_at, '') = json_extract(r.value, '$.hi_at') AND photos.id <= json_extract(r.value, '$.hi_id')))))
		AND id NOT IN (SELECT value FROM json_each(?))`
	args = []any{enc(months), enc(days), idList(s.IncludeIDs), enc(spans), idList(s.ExcludeIDs)}
	return where, args, true, nil
}

// resolveRanges looks up both endpoints of every range and orders each pair grid-wise
// (lower key first), so the caller's order of endpoints does not matter.
func resolveRanges(ctx context.Context, db *sql.DB, in [][2]int64) ([][2]rangeEnd, error) {
	if len(in) == 0 {
		return nil, nil
	}
	want := make([]int64, 0, 2*len(in))
	for _, r := range in {
		want = append(want, r[0], r[1])
	}
	b, _ := json.Marshal(want)
	rows, err := db.QueryContext(ctx, `SELECT id, COALESCE(taken_at, '') FROM photos
		WHERE id IN (SELECT value FROM json_each(?))`, string(b))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	at := map[int64]string{}
	for rows.Next() {
		var id int64
		var a string
		if err := rows.Scan(&id, &a); err != nil {
			return nil, err
		}
		at[id] = a
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	out := make([][2]rangeEnd, 0, len(in))
	for _, r := range in {
		a, okA := at[r[0]]
		z, okZ := at[r[1]]
		if !okA || !okZ {
			return nil, invalidf("range %d..%d names an unknown photo id", r[0], r[1])
		}
		lo, hi := rangeEnd{a, r[0]}, rangeEnd{z, r[1]}
		if lo.At > hi.At || (lo.At == hi.At && lo.ID > hi.ID) {
			lo, hi = hi, lo
		}
		out = append(out, [2]rangeEnd{lo, hi})
	}
	return out, nil
}

// ResolveSelection returns the selected files in grid order (taken_at DESC, id DESC).
// Validation problems wrap ErrInvalidSelection; anything else is a database error.
func ResolveSelection(ctx context.Context, db *sql.DB, s Selection) ([]Resolved, error) {
	where, args, ok, err := selectionWhere(ctx, db, s)
	if err != nil {
		return nil, err
	}
	out := []Resolved{}
	if !ok {
		return out, nil
	}
	rows, err := db.QueryContext(ctx, `SELECT id, rel_path FROM photos WHERE `+where+
		` ORDER BY COALESCE(taken_at, '') DESC, id DESC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var r Resolved
		if err := rows.Scan(&r.ID, &r.RelPath); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// CountSelection counts what ResolveSelection would return, without loading the files.
func CountSelection(ctx context.Context, db *sql.DB, s Selection) (int, error) {
	where, args, ok, err := selectionWhere(ctx, db, s)
	if err != nil || !ok {
		return 0, err
	}
	var n int
	err = db.QueryRowContext(ctx, `SELECT count(*) FROM photos WHERE `+where, args...).Scan(&n)
	return n, err
}
