package exif

import (
	"context"
	"strings"
	"testing"

	"photocurator/internal/metadata"
	"photocurator/internal/testsupport"
)

func newWriter(t *testing.T) *Writer {
	t.Helper()
	cfg, err := metadata.WriteConfig(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	w := &Writer{ConfigPath: cfg}
	t.Cleanup(func() { w.Close() })
	return w
}

func TestWriterRejectsLineBreaksInArguments(t *testing.T) {
	w := newWriter(t)
	path := testsupport.CopyFixture(t, "library/2024/07/sample.jpg")
	ctx := context.Background()

	if err := w.Write(ctx, path, []string{"-XMP-dc:Subject=a\n-delete_original"}); err == nil {
		t.Error("argument with line break was accepted")
	}
	if err := w.Write(ctx, path, []string{"-XMP-dc:Subject=a\r"}); err == nil {
		t.Error("argument with carriage return was accepted")
	}
	if err := w.Write(ctx, path+"\n-overwrite_original", nil); err == nil {
		t.Error("path with line break was accepted")
	}
	if _, err := w.ImageHash(ctx, path+"\r\n"); err == nil {
		t.Error("hash path with line break was accepted")
	}
}

func TestImageHashIgnoresAMetadataChange(t *testing.T) {
	w := newWriter(t)
	path := testsupport.CopyFixture(t, "library/2024/07/sample.jpg")
	ctx := context.Background()

	before, err := w.ImageHash(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	if len(before) != 64 {
		t.Fatalf("hash %q is not a SHA-256 hex digest", before)
	}
	if err := w.Write(ctx, path, []string{"-XMP-dc:Subject=geaendert"}); err != nil {
		t.Fatal(err)
	}
	after, err := w.ImageHash(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	if before != after {
		t.Errorf("image data hash changed: %s -> %s", before, after)
	}
}

func TestWriteReturnsExifToolErrorText(t *testing.T) {
	w := newWriter(t)
	missing := testsupport.CopyFixture(t, "library/2024/07/sample.jpg") + ".fehlt"

	err := w.Write(context.Background(), missing, []string{"-XMP-dc:Subject=x"})
	if err == nil || !strings.Contains(err.Error(), "File not found") {
		t.Fatalf("want ExifTool error text, got %v", err)
	}
	// The session survives a reported error.
	if _, err := w.ImageHash(context.Background(), testsupport.CopyFixture(t, "library/2024/07/sample.jpg")); err != nil {
		t.Fatal(err)
	}
}
