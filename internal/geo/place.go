// Package geo turns Nominatim answers into the places the app shows and stores.
package geo

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// MaxResults caps a search answer; Nominatim is asked for the same limit.
const MaxResults = 10

// Parts are the address pieces written to the photo's place fields.
type Parts struct {
	Sublocation string `json:"sublocation"`
	City        string `json:"city"`
	State       string `json:"state"`
	Country     string `json:"country"`
	CountryCode string `json:"countryCode"`
}

// Place is one geocoding hit. Label is what the user sees: "<name>, <town>".
type Place struct {
	Name    string  `json:"name"`
	Address string  `json:"address"`
	Kind    string  `json:"kind"`
	Lat     float64 `json:"lat"`
	Lon     float64 `json:"lon"`
	Label   string  `json:"label"`
	Parts   Parts   `json:"parts"`
}

type hit struct {
	Lat         string            `json:"lat"`
	Lon         string            `json:"lon"`
	Type        string            `json:"type"`
	Name        string            `json:"name"`
	DisplayName string            `json:"display_name"`
	NameDetails map[string]string `json:"namedetails"`
	Address     map[string]string `json:"address"`
	Error       string            `json:"error"`
}

// ParseSearch maps a jsonv2 search answer, keeping Nominatim's order, to at most MaxResults places.
func ParseSearch(data []byte) ([]Place, error) {
	var hits []hit
	if err := json.Unmarshal(data, &hits); err != nil {
		return nil, fmt.Errorf("geo: parse search answer: %w", err)
	}
	if len(hits) > MaxResults {
		hits = hits[:MaxResults]
	}
	places := make([]Place, 0, len(hits))
	for _, h := range hits {
		p, err := h.place()
		if err != nil {
			return nil, err
		}
		places = append(places, p)
	}
	return places, nil
}

// ParseReverse maps a jsonv2 reverse answer. Nominatim answers 200 with an error member when
// nothing is found there; that is no place and no failure.
func ParseReverse(data []byte) (*Place, error) {
	var h hit
	if err := json.Unmarshal(data, &h); err != nil {
		return nil, fmt.Errorf("geo: parse reverse answer: %w", err)
	}
	if h.Error != "" {
		return nil, nil
	}
	p, err := h.place()
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (h hit) place() (Place, error) {
	// jsonv2 sends coordinates as strings.
	lat, err := strconv.ParseFloat(h.Lat, 64)
	if err != nil {
		return Place{}, fmt.Errorf("geo: bad lat %q: %w", h.Lat, err)
	}
	lon, err := strconv.ParseFloat(h.Lon, 64)
	if err != nil {
		return Place{}, fmt.Errorf("geo: bad lon %q: %w", h.Lon, err)
	}
	name := h.Name
	if name == "" {
		name = h.NameDetails["name"]
	}
	town := firstNonEmpty(h.Address, "city", "town", "village", "municipality")

	label := name + ", " + town
	switch {
	case name == "" && town == "":
		label, _, _ = strings.Cut(h.DisplayName, ",")
		label = strings.TrimSpace(label)
	case name == "" || name == town:
		label = town
	case town == "":
		label = name
	}

	sub := name
	if name == town {
		sub = firstNonEmpty(h.Address, "suburb", "neighbourhood", "hamlet")
	}
	return Place{
		Name:    name,
		Address: h.DisplayName,
		Kind:    h.Type,
		Lat:     lat,
		Lon:     lon,
		Label:   label,
		Parts: Parts{
			Sublocation: sub,
			City:        town,
			State:       h.Address["state"],
			Country:     h.Address["country"],
			CountryCode: strings.ToUpper(h.Address["country_code"]),
		},
	}, nil
}

func firstNonEmpty(m map[string]string, keys ...string) string {
	for _, k := range keys {
		if m[k] != "" {
			return m[k]
		}
	}
	return ""
}
