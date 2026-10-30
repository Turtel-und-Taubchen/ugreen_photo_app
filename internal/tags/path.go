// Package tags validates hierarchical tag paths and builds the tag tree.
package tags

import (
	"fmt"
	"strings"
	"unicode"
)

// MaxSegmentBytes is the IPTC limit for one keyword; a segment is stored as one.
const MaxSegmentBytes = 64

// ParsePath returns the canonical form of a tag path: segments split on "/",
// each trimmed of surrounding whitespace, rejoined with "/". It does not
// normalise Unicode or fold case.
func ParsePath(input string) (string, error) {
	if strings.TrimSpace(input) == "" {
		return "", fmt.Errorf("Der Tag-Name darf nicht leer sein.")
	}
	segs := strings.Split(input, "/")
	for i, s := range segs {
		s = strings.TrimSpace(s)
		segs[i] = s
		if s == "" {
			return "", fmt.Errorf("Der Tag-Name \"%s\" enthält ein leeres Segment (Position %d); führende, abschließende oder doppelte \"/\" sind nicht erlaubt.", input, i+1)
		}
		if c := strings.IndexAny(s, "|;,"); c >= 0 {
			return "", fmt.Errorf("Das Segment \"%s\" enthält das Zeichen \"%c\"; es dient in den Metadatenfeldern als Trennzeichen und ist nicht erlaubt.", s, s[c])
		}
		for _, r := range s {
			if unicode.IsControl(r) {
				return "", fmt.Errorf("Das Segment \"%s\" enthält ein Steuerzeichen (U+%04X); das ist nicht erlaubt.", s, r)
			}
		}
		if len(s) > MaxSegmentBytes {
			return "", fmt.Errorf("Das Segment \"%s\" ist %d Byte lang, erlaubt sind höchstens %d Byte (IPTC); Umlaute zählen 2 Byte.", s, len(s), MaxSegmentBytes)
		}
	}
	return strings.Join(segs, "/"), nil
}
