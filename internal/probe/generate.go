package probe

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const captureStamp = "2001:01:01 12:00:00"

// CaptureTime is the capture date of every probe file, also used as its mtime.
func CaptureTime() time.Time {
	return time.Date(2001, 1, 1, 12, 0, 0, 0, time.Local)
}

// Generate writes the probe files into outDir, which must not exist or be empty. The steps
// run in order: images, videos, mutate inputs, protocol.
func Generate(outDir string) error {
	entries, err := os.ReadDir(outDir)
	if err == nil && len(entries) > 0 {
		return fmt.Errorf("output directory %s is not empty", outDir)
	}
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	for _, step := range []func(string) error{generateImages, generateVideos, generateMutateFiles, generateProtocol} {
		if err := step(outDir); err != nil {
			return err
		}
	}
	return nil
}

func generateImages(outDir string) error {
	for _, p := range probes {
		switch p.Format {
		case FormatJPEG, FormatHEIC, FormatPNG:
		default:
			continue
		}
		path := filepath.Join(outDir, p.File)
		if err := WriteImage(path, p.Format, RenderText(p.PixelText())); err != nil {
			return fmt.Errorf("%s: %w", p.File, err)
		}
		if err := WriteTags(path, imageAssignments(p)); err != nil {
			return err
		}
		if err := os.Chtimes(path, CaptureTime(), CaptureTime()); err != nil {
			return err
		}
	}
	return nil
}

func imageAssignments(p Probe) []TagValue {
	if p.FieldKey == KeyNoMeta {
		return []TagValue{{"all", ""}}
	}
	a := []TagValue{{"EXIF:DateTimeOriginal", captureStamp}, {"EXIF:CreateDate", captureStamp}}
	a = append(a, p.Fixed...)
	a = append(a, p.Texts...)
	return append(a, p.GPS...)
}

func generateMutateFiles(outDir string) error {
	for _, f := range MutateFiles() {
		path := filepath.Join(outDir, f.File)
		if err := WriteImage(path, FormatJPEG, RenderText(f.Marker)); err != nil {
			return fmt.Errorf("%s: %w", f.File, err)
		}
		a := []TagValue{{"EXIF:DateTimeOriginal", captureStamp}, {"EXIF:CreateDate", captureStamp}, {"XMP-dc:Subject", f.Marker}}
		if err := WriteTags(path, a); err != nil {
			return err
		}
		if err := os.Chtimes(path, CaptureTime(), CaptureTime()); err != nil {
			return err
		}
	}
	return nil
}
