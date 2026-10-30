package tags

import (
	"strings"
	"testing"
)

func TestInvalidTagNamesAreRejectedWithAReason(t *testing.T) {
	cases := []struct{ name, in, want string }{
		{"empty", "", "leer"},
		{"blank", "   ", "leer"},
		{"double slash", "Anlass//X", "leeres Segment"},
		{"leading slash", "/Anlass", "leeres Segment"},
		{"trailing slash", "Anlass/", "leeres Segment"},
		{"pipe", "A|B", "|"},
		{"semicolon", "A;B", ";"},
		{"comma", "A,B", ","},
		{"newline", "A\nB", "Steuerzeichen"},
		{"tab", "A\tB", "Steuerzeichen"},
		{"65 bytes", strings.Repeat("a", 65), "höchstens 64 Byte (IPTC)"},
		{"33 umlauts", strings.Repeat("ä", 33), "höchstens 64 Byte (IPTC)"},
	}
	for _, c := range cases {
		_, err := ParsePath(c.in)
		if err == nil {
			t.Errorf("%s: accepted %q", c.name, c.in)
			continue
		}
		if !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: error %q does not mention %q", c.name, err, c.want)
		}
	}
	if _, err := ParsePath("A|B"); err == nil || !strings.Contains(err.Error(), "Trennzeichen") {
		t.Errorf("separator reason missing: %v", err)
	}
}

func TestValidTagNamesAreAcceptedInCanonicalForm(t *testing.T) {
	cases := []struct{ in, want string }{
		{"Wir zwei", "Wir zwei"},
		{"Reise/Österreich", "Reise/Österreich"},
		{strings.Repeat("a", 64), strings.Repeat("a", 64)},
		{strings.Repeat("ä", 32), strings.Repeat("ä", 32)},
		{" Anlass / Geburtstag ", "Anlass/Geburtstag"},
	}
	for _, c := range cases {
		got, err := ParsePath(c.in)
		if err != nil || got != c.want {
			t.Errorf("ParsePath(%q) = %q, %v; want %q", c.in, got, err, c.want)
		}
	}
}
