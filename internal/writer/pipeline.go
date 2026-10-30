// Package writer changes metadata in library files so that an original is never left damaged:
// every write is guarded by a pre-copy, verified by a read-back and undone in place on failure.
package writer

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"photocurator/internal/exif"
	"photocurator/internal/library"
	"photocurator/internal/metadata"
)

// State is the outcome of one file.
type State string

const (
	StateOK      State = "ok"
	StateSkipped State = "skipped" // nothing was touched
	StateFailed  State = "failed"  // the original holds its previous bytes again
)

// Result is the outcome of WriteFile; Reason is user-facing text.
type Result struct {
	State  State
	Reason string
	// Old is read before the write and New is the read-back, never the requested values. Only
	// the fields the change touches are set, and only when the file was read successfully.
	Old, New Journal
	// WrittenAt is zero when the file needed no write.
	WrittenAt time.Time
}

const (
	reasonOutside  = "außerhalb von /photos"
	reasonNotFound = "Datei nicht gefunden"
	reasonCurrent  = "bereits aktuell"
)

// Pipeline writes metadata changes into files below Root, one file at a time.
type Pipeline struct {
	Root      string
	BackupDir string // the in-flight pre-copies live below it, never below Root
	Exif      *exif.Writer
	Reader    exif.Reader // reads metadata.ReadTags with the same config as Exif
	Index     *library.Scanner
	Quiet     time.Duration    // files modified more recently than this are skipped
	Now       func() time.Time // nil means time.Now
	// Backups keeps the dated retention copy taken before a file is first changed; nil means
	// none. It is never a restore source: a dated copy can predate later jobs on the same file.
	Backups *Backups
	// AfterWrite runs between the ExifTool write and the read-back (tests only). An error
	// stands for a crash: WriteFile returns at once and leaves the pre-copy for Recover.
	AfterWrite func(abs string) error
}

func (p *Pipeline) now() time.Time {
	if p.Now != nil {
		return p.Now()
	}
	return time.Now()
}

// resolve maps rel to the file below Root. reason is set when rel leaves Root, names no
// regular file, or passes through a symlink.
func (p *Pipeline) resolve(rel string) (abs, cleanRel string, info fs.FileInfo, reason string) {
	r := filepath.Clean(filepath.FromSlash(rel))
	if filepath.IsAbs(r) || r == "." || r == ".." || strings.HasPrefix(r, ".."+string(filepath.Separator)) {
		return "", "", nil, reasonOutside
	}
	root, err := filepath.EvalSymlinks(p.Root)
	if err != nil {
		return "", "", nil, "Bibliothek nicht erreichbar: " + err.Error()
	}
	abs = filepath.Join(root, r)
	info, err = os.Lstat(abs)
	if errors.Is(err, fs.ErrNotExist) {
		return "", "", nil, reasonNotFound
	}
	if err != nil {
		return "", "", nil, "Datei nicht lesbar: " + err.Error()
	}
	if !info.Mode().IsRegular() {
		return "", "", nil, reasonOutside
	}
	// A symlinked folder on the way could lead out of Root; the scan does not follow one either.
	if resolved, err := filepath.EvalSymlinks(abs); err != nil || resolved != abs {
		return "", "", nil, reasonOutside
	}
	return abs, filepath.ToSlash(r), info, ""
}

// WriteFile applies c to the file rel ('/'-separated, relative to Root).
//
// Order is what keeps the original safe: the dated backup exists before anything else is
// written for the file, the durable pre-copy exists before ExifTool touches the file, and the
// pre-copy is deleted only after the read-back and the image hash confirmed the write or the
// original was restored from it. A restore rewrites the original through its own inode, and
// its source is always this write's pre-copy, never the dated backup.
func (p *Pipeline) WriteFile(ctx context.Context, rel string, c metadata.Change) Result {
	abs, rel, info, reason := p.resolve(rel)
	if reason == reasonNotFound {
		// The caller asked for a write that cannot happen, unlike a file that is merely off limits.
		return Result{State: StateFailed, Reason: reason}
	}
	if reason != "" {
		return Result{State: StateSkipped, Reason: reason}
	}
	format, ok := library.FormatOf(abs)
	if !ok {
		return Result{State: StateSkipped, Reason: "Format nicht unterstützt"}
	}
	now := p.now()
	if age := now.Sub(info.ModTime()); p.Quiet > 0 && age < p.Quiet && age > -p.Quiet {
		return Result{State: StateSkipped, Reason: "Ruhezeit"}
	}
	if pending, err := p.pendingCopy(rel); err != nil {
		return Result{State: StateFailed, Reason: "Vorab-Kopie nicht prüfbar: " + err.Error()}
	} else if pending != "" {
		// Overwriting it could replace the only good copy with a half-written original.
		return Result{State: StateFailed, Reason: "unterbrochener Schreibvorgang wartet auf Wiederherstellung"}
	}
	if err := ctx.Err(); err != nil {
		return Result{State: StateSkipped, Reason: "abgebrochen"}
	}
	// Read and diff before any backup or pre-copy: a file that is already current must stay
	// untouched, with no retention copy taken for a write that never happens.
	work := context.WithoutCancel(ctx)
	old, fields, err := p.readMetaFields(work, abs)
	if err != nil {
		return Result{State: StateFailed, Reason: "Metadaten nicht lesbar: " + err.Error()}
	}
	args, err := metadata.Args(format, fields, c)
	if err != nil {
		return Result{State: StateFailed, Reason: err.Error()}
	}
	if len(args) == 0 {
		return Result{State: StateSkipped, Reason: reasonCurrent}
	}

	// After the pending-copy check: with a copy waiting, the original may be half-written and
	// must not become the retention backup.
	if p.Backups != nil {
		if _, _, err := p.Backups.Ensure(rel); err != nil {
			return Result{State: StateFailed, Reason: "Backup fehlgeschlagen: " + err.Error()}
		}
	}

	copyPath := filepath.Join(p.inflightRoot(), now.Format("2006-01-02"), filepath.FromSlash(rel))
	if err := preCopy(copyPath, abs, info.ModTime()); err != nil {
		return Result{State: StateFailed, Reason: "Vorab-Kopie fehlgeschlagen: " + err.Error()}
	}
	// From here on a cancelled ctx must not leave the file between write and verification.
	unchanged := func(reason string) Result {
		if err := p.dropCopy(copyPath); err != nil {
			reason += "; Vorab-Kopie nicht gelöscht: " + err.Error()
		}
		return Result{State: StateFailed, Reason: reason}
	}
	restore := func(reason string) Result {
		// An untouched original (ExifTool failed before changing it) needs no restore, and a
		// read-only one could not take it; the copy is then simply dropped.
		if same, err := sameBytes(abs, copyPath); err == nil && same {
			return unchanged(reason)
		}
		if err := restoreInPlace(abs, copyPath); err != nil {
			return Result{State: StateFailed, Reason: reason + "; Wiederherstellung fehlgeschlagen, Vorab-Kopie bleibt: " + err.Error()}
		}
		return unchanged(reason + "; Original wiederhergestellt")
	}

	hash, err := p.Exif.ImageHash(work, abs)
	if err != nil {
		return unchanged("Bilddaten-Prüfsumme nicht ermittelbar: " + err.Error())
	}

	if err := p.Exif.Write(work, abs, args); err != nil {
		return restore("ExifTool: " + err.Error())
	}
	if p.AfterWrite != nil {
		if err := p.AfterWrite(abs); err != nil {
			return Result{State: StateFailed, Reason: err.Error()}
		}
	}

	got, err := p.readMeta(work, abs)
	if err != nil {
		return restore("Read-back nicht lesbar: " + err.Error())
	}
	if !metadata.Matches(format, got, metadata.Expect(old, c)) {
		return restore("Read-back weicht ab")
	}
	if after, err := p.Exif.ImageHash(work, abs); err != nil {
		return restore("Bilddaten-Prüfsumme nach dem Schreiben nicht ermittelbar: " + err.Error())
	} else if after != hash {
		return restore("Bilddaten verändert")
	}

	affected := AffectedBy(c)
	res := Result{State: StateOK, Old: JournalOf(old, affected), New: JournalOf(got, affected), WrittenAt: now}
	if err := p.dropCopy(copyPath); err != nil {
		// The write is verified, but Recover would roll it back while the copy exists.
		res = Result{State: StateFailed, Reason: "geschrieben, aber Vorab-Kopie nicht gelöscht: " + err.Error()}
	}
	if p.Index != nil {
		if err := p.Index.IndexFiles(work, []string{rel}); err != nil {
			res.Reason = strings.TrimPrefix(res.Reason+"; Index nicht aktualisiert: "+err.Error(), "; ")
		}
	}
	return res
}

func (p *Pipeline) readMeta(ctx context.Context, abs string) (metadata.Meta, error) {
	m, _, err := p.readMetaFields(ctx, abs)
	return m, err
}

// readMetaFields also returns the raw ExifTool record, which tag writing diffs per field.
func (p *Pipeline) readMetaFields(ctx context.Context, abs string) (metadata.Meta, map[string]any, error) {
	recs, err := p.Reader.Read(ctx, []string{abs})
	if err != nil {
		return metadata.Meta{}, nil, err
	}
	rec, ok := recs[abs]
	if !ok {
		return metadata.Meta{}, nil, errors.New("kein Ergebnis von ExifTool")
	}
	if rec.Err != "" {
		return metadata.Meta{}, nil, errors.New(rec.Err)
	}
	return metadata.Parse(rec.Fields), rec.Fields, nil
}
