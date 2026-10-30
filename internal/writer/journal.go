package writer

import (
	"encoding/json"
	"slices"

	"photocurator/internal/metadata"
)

// JournalLocation is a place in domain terms. Meta carries no country code today, so it stays
// empty until a source for it exists.
type JournalLocation struct {
	Lat         *float64 `json:"lat"`
	Lon         *float64 `json:"lon"`
	Label       string   `json:"label"`
	Sublocation string   `json:"sublocation"`
	City        string   `json:"city"`
	State       string   `json:"state"`
	Country     string   `json:"country"`
	CountryCode string   `json:"countryCode"`
}

// Field is one journalled value. Affected tells "the action touches this field" apart from
// "the field is empty": an affected empty field has a nil Value and is stored as null.
type Field[T any] struct {
	Affected bool
	Value    *T
}

// Affected names the fields a Change touches; only those are journalled.
type Affected struct{ Tags, Location, ReviewState bool }

// AffectedBy returns the fields c touches.
func AffectedBy(c metadata.Change) Affected {
	return Affected{
		Tags:        len(c.AddTags) > 0 || len(c.RemoveTags) > 0,
		Location:    c.Location != nil || c.ClearLocation,
		ReviewState: c.ReviewState != "" || c.ClearReviewState,
	}
}

// Journal holds the values of the affected fields of one file, as read from the file. Later
// stories compare it with what a file holds now, so both sides must come from JournalOf.
type Journal struct {
	Tags        Field[[]string] // sorted canonical "Parent/Child" paths
	Location    Field[JournalLocation]
	ReviewState Field[string]
}

// JournalOf normalises m into the journal shape for the given fields.
func JournalOf(m metadata.Meta, a Affected) Journal {
	var j Journal
	if a.Tags {
		j.Tags.Affected = true
		if len(m.Tags) > 0 {
			tags := slices.Clone(m.Tags)
			slices.Sort(tags)
			j.Tags.Value = &tags
		}
	}
	if a.Location {
		j.Location.Affected = true
		loc := JournalLocation{Lat: m.Lat, Lon: m.Lon, Label: m.Label, Sublocation: m.Sublocation,
			City: m.City, State: m.State, Country: m.Country}
		if loc != (JournalLocation{}) {
			j.Location.Value = &loc
		}
	}
	if a.ReviewState {
		j.ReviewState.Affected = true
		if m.ReviewState != "" {
			j.ReviewState.Value = &m.ReviewState
		}
	}
	return j
}

// IsZero reports whether no field is affected, i.e. there is nothing to store.
func (j Journal) IsZero() bool {
	return !j.Tags.Affected && !j.Location.Affected && !j.ReviewState.Affected
}

// MarshalJSON leaves unaffected fields out and writes affected empty ones as null.
func (j Journal) MarshalJSON() ([]byte, error) {
	m := map[string]any{}
	if j.Tags.Affected {
		m["tags"] = j.Tags.Value
	}
	if j.Location.Affected {
		m["location"] = j.Location.Value
	}
	if j.ReviewState.Affected {
		m["reviewState"] = j.ReviewState.Value
	}
	return json.Marshal(m)
}

// UnmarshalJSON is the inverse of MarshalJSON.
func (j *Journal) UnmarshalJSON(b []byte) error {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(b, &m); err != nil {
		return err
	}
	*j = Journal{}
	if err := decodeField(m, "tags", &j.Tags); err != nil {
		return err
	}
	if err := decodeField(m, "location", &j.Location); err != nil {
		return err
	}
	return decodeField(m, "reviewState", &j.ReviewState)
}

func decodeField[T any](m map[string]json.RawMessage, key string, f *Field[T]) error {
	raw, ok := m[key]
	if !ok {
		return nil
	}
	f.Affected = true
	return json.Unmarshal(raw, &f.Value) // null leaves Value nil
}

// journalJSON is the value stored in a job_items column; nil (SQL NULL) when nothing is affected.
func journalJSON(j Journal) (any, error) {
	if j.IsZero() {
		return nil, nil
	}
	b, err := json.Marshal(j)
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

func parseJournal(s *string) (Journal, error) {
	var j Journal
	if s == nil {
		return j, nil
	}
	err := json.Unmarshal([]byte(*s), &j)
	return j, err
}
