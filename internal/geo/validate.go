package geo

import (
	"math"
	"strings"
	"unicode"
	"unicode/utf8"
)

const maxTextRunes = 200

// LocationInputError is a user-facing (German) reason why a place input was refused.
type LocationInputError struct{ Msg string }

func (e *LocationInputError) Error() string { return e.Msg }

// ValidatePoint accepts finite coordinates on the globe.
func ValidatePoint(lat, lon float64) error {
	if math.IsNaN(lat) || math.IsInf(lat, 0) || math.IsNaN(lon) || math.IsInf(lon, 0) {
		return &LocationInputError{"Koordinaten müssen Zahlen sein"}
	}
	if lat < -90 || lat > 90 {
		return &LocationInputError{"Breitengrad muss zwischen -90 und 90 liegen"}
	}
	if lon < -180 || lon > 180 {
		return &LocationInputError{"Längengrad muss zwischen -180 und 180 liegen"}
	}
	return nil
}

// ValidateLocationText trims and checks the place name and address parts and returns the
// cleaned values; CountryCode is upper-cased. Empty parts are allowed (they mean "no value").
func ValidateLocationText(name string, parts Parts) (string, Parts, error) {
	name, err := cleanText("Ortsname", name)
	if err != nil {
		return "", Parts{}, err
	}
	if name == "" {
		return "", Parts{}, &LocationInputError{"Ortsname darf nicht leer sein"}
	}
	for _, f := range []struct {
		label string
		v     *string
	}{
		{"Ortsteil", &parts.Sublocation}, {"Stadt", &parts.City}, {"Bundesland", &parts.State},
		{"Land", &parts.Country}, {"Ländercode", &parts.CountryCode},
	} {
		if *f.v, err = cleanText(f.label, *f.v); err != nil {
			return "", Parts{}, err
		}
	}
	parts.CountryCode = strings.ToUpper(parts.CountryCode)
	return name, parts, nil
}

func cleanText(label, s string) (string, error) {
	s = strings.TrimSpace(s)
	if utf8.RuneCountInString(s) > maxTextRunes {
		return "", &LocationInputError{label + " darf höchstens 200 Zeichen lang sein"}
	}
	if strings.IndexFunc(s, unicode.IsControl) >= 0 {
		return "", &LocationInputError{label + " enthält Steuerzeichen"}
	}
	return s, nil
}
