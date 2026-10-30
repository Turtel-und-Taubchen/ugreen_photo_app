package library

import (
	"testing"
	"time"
	_ "time/tzdata"

	"photocurator/internal/metadata"
)

func TestTheTakenDateFallsBackFromEXIFToQuickTimeToTheFileNameToTheModificationTime(t *testing.T) {
	vienna, err := time.LoadLocation("Europe/Vienna")
	if err != nil {
		t.Fatal(err)
	}
	mtime := time.Date(2023, 1, 5, 23, 30, 0, 0, time.UTC) // 00:30 next day in Vienna

	tests := []struct {
		name       string
		meta       metadata.Meta
		file       string
		wantAt     string
		wantSource string
	}{
		{"exif wins and is kept as written", metadata.Meta{DateTimeOriginal: "2024:07:14 10:00:00", QuickTimeCreate: "2024:08:20 08:00:00"}, "IMG-20200101-WA0001.jpg", "2024-07-14T10:00:00", "exif"},
		{"exif ignores a zone suffix", metadata.Meta{DateTimeOriginal: "2024:07:14 10:00:00+02:00"}, "a.jpg", "2024-07-14T10:00:00", "exif"},
		{"zero exif date falls to quicktime", metadata.Meta{DateTimeOriginal: "0000:00:00 00:00:00", QuickTimeCreate: "2024:08:20 08:00:00"}, "a.mp4", "2024-08-20T10:00:00", "quicktime"},
		{"quicktime UTC to Vienna in summer", metadata.Meta{QuickTimeCreate: "2024:08:20 22:30:00"}, "a.mp4", "2024-08-21T00:30:00", "quicktime"},
		{"quicktime UTC to Vienna in winter", metadata.Meta{QuickTimeCreate: "2024:01:20 22:30:00"}, "a.mp4", "2024-01-20T23:30:00", "quicktime"},
		{"zero quicktime falls to file name", metadata.Meta{QuickTimeCreate: "0000:00:00 00:00:00"}, "IMG-20240102-WA0003.jpg", "2024-01-02T00:00:00", "filename"},
		{"video chat name", metadata.Meta{}, "VID-20240315-WA0001.mp4", "2024-03-15T00:00:00", "filename"},
		{"stamped name", metadata.Meta{}, "PXL_20240315_142530.jpg", "2024-03-15T14:25:30", "filename"},
		{"invalid stamped name falls to mtime", metadata.Meta{}, "20241341_250000.jpg", "2023-01-06T00:30:00", "mtime"},
		{"invalid chat date falls to mtime", metadata.Meta{}, "IMG-20241341-WA0001.jpg", "2023-01-06T00:30:00", "mtime"},
		{"nothing else uses mtime in the zone", metadata.Meta{}, "holiday.jpg", "2023-01-06T00:30:00", "mtime"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			at, src := TakenDate(tc.meta, tc.file, mtime, vienna)
			if at != tc.wantAt || src != tc.wantSource {
				t.Errorf("got %s %s, want %s %s", at, src, tc.wantAt, tc.wantSource)
			}
		})
	}
}
