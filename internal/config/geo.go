package config

import "strings"

const defaultNominatimURL = "https://nominatim.openstreetmap.org"

// Geo configures the place search. An empty Contact is valid: Nominatim's policy requires a
// contact, so the search is switched off instead of failing the start.
type Geo struct {
	Contact string
	BaseURL string
}

// GeoFromEnv reads PC_NOMINATIM_CONTACT and PC_NOMINATIM_URL.
func GeoFromEnv(get func(string) string) Geo {
	base := strings.TrimRight(strings.TrimSpace(get("PC_NOMINATIM_URL")), "/")
	if base == "" {
		base = defaultNominatimURL
	}
	return Geo{Contact: strings.TrimSpace(get("PC_NOMINATIM_CONTACT")), BaseURL: base}
}
