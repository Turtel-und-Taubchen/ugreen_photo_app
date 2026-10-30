// Package library holds pure rules about the photo library's files.
package library

import (
	"regexp"
	"time"

	"photocurator/internal/metadata"
)

const outFormat = "2006-01-02T15:04:05"

var (
	whatsApp = regexp.MustCompile(`^(?:IMG|VID)-(\d{8})-WA`)
	stamped  = regexp.MustCompile(`(?:^|\D)(\d{8})_(\d{6})(?:\D|$)`)
)

// parseExifDate reads "YYYY:MM:DD HH:MM:SS"; any zone or subsecond suffix is ignored.
// Zero dates fail the calendar check and so count as absent.
func parseExifDate(s string) (time.Time, bool) {
	if len(s) < 19 {
		return time.Time{}, false
	}
	t, err := time.Parse("2006:01:02 15:04:05", s[:19])
	return t, err == nil
}

func dateFromName(name string) (string, bool) {
	if m := whatsApp.FindStringSubmatch(name); m != nil {
		if t, err := time.Parse("20060102", m[1]); err == nil {
			return t.Format(outFormat), true
		}
	}
	for _, m := range stamped.FindAllStringSubmatch(name, -1) {
		if t, err := time.Parse("20060102150405", m[1]+m[2]); err == nil {
			return t.Format(outFormat), true
		}
	}
	return "", false
}

// TakenDate picks the capture time: EXIF wall-clock time as written, QuickTime (UTC by
// spec) converted to loc, a date in the file name, else the modification time in loc.
func TakenDate(m metadata.Meta, fileName string, mtime time.Time, loc *time.Location) (takenAt, source string) {
	if loc == nil {
		loc = time.UTC
	}
	if t, ok := parseExifDate(m.DateTimeOriginal); ok {
		return t.Format(outFormat), "exif"
	}
	if t, ok := parseExifDate(m.QuickTimeCreate); ok {
		return t.In(loc).Format(outFormat), "quicktime"
	}
	if s, ok := dateFromName(fileName); ok {
		return s, "filename"
	}
	return mtime.In(loc).Format(outFormat), "mtime"
}
