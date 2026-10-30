package geo

import (
	"errors"
	"math"
	"strings"
	"testing"
)

func TestLocationInputWithBadCoordinatesNamesOrControlCharsIsRefused(t *testing.T) {
	for name, p := range map[string][2]float64{
		"lat too high": {90.1, 0}, "lat too low": {-91, 0}, "lon too high": {0, 180.5},
		"lon too low": {0, -181}, "nan": {math.NaN(), 0}, "inf": {0, math.Inf(1)},
	} {
		var le *LocationInputError
		if err := ValidatePoint(p[0], p[1]); !errors.As(err, &le) || le.Msg == "" {
			t.Errorf("%s: err = %v, want a LocationInputError", name, err)
		}
	}
	for _, p := range [][2]float64{{90, 180}, {-90, -180}, {0, 0}} {
		if err := ValidatePoint(p[0], p[1]); err != nil {
			t.Errorf("%v: %v, want accepted", p, err)
		}
	}

	long := strings.Repeat("ä", 201)
	for name, c := range map[string]struct {
		name  string
		parts Parts
	}{
		"empty name":      {"   ", Parts{}},
		"long name":       {long, Parts{}},
		"control in name": {"Am\x00See", Parts{}},
		"newline in city": {"See", Parts{City: "Wi\nen"}},
		"tab inside":      {"See", Parts{State: "Ober\töster"}},
		"long country":    {"See", Parts{Country: long}},
	} {
		var le *LocationInputError
		if _, _, err := ValidateLocationText(c.name, c.parts); !errors.As(err, &le) || le.Msg == "" {
			t.Errorf("%s: err = %v, want a LocationInputError", name, err)
		}
	}

	got, parts, err := ValidateLocationText("  Stephansdom ", Parts{City: " Wien ", CountryCode: "at"})
	if err != nil || got != "Stephansdom" || parts.City != "Wien" || parts.CountryCode != "AT" {
		t.Errorf("got %q %+v, %v; want trimmed name and parts with upper-cased code", got, parts, err)
	}
	if _, _, err := ValidateLocationText(strings.Repeat("ä", 200), Parts{}); err != nil {
		t.Errorf("200 runes: %v, want accepted", err)
	}
}
