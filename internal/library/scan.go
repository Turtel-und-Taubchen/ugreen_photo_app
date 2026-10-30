package library

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"photocurator/internal/exif"
	"photocurator/internal/metadata"
)

// blockSize is the number of files per ExifTool call and per index transaction.
const blockSize = 200

// Scanner brings index.db in line with the files below Root. It never writes below Root:
// files are only listed and stat'ed here, and read by the Reader.
type Scanner struct {
	Root   string
	DB     *sql.DB
	Reader interface {
		Read(ctx context.Context, paths []string) (map[string]exif.Record, error)
	}
	Quiet time.Duration    // files modified more recently than this are left for a later scan
	Now   func() time.Time // nil means time.Now
	Loc   *time.Location   // zone for QuickTime and mtime dates; nil means UTC
}

type indexedFile struct {
	size, mtimeNS int64
	failed        bool
}

// Run reads new, changed and previously unreadable files into the index and drops rows whose
// file is gone. progress, if set, is called after each block with the files read so far and
// the number to read. A cancelled ctx stops the scan after the block in progress.
func (s *Scanner) Run(ctx context.Context, progress func(done, total int)) (Result, error) {
	now := time.Now
	if s.Now != nil {
		now = s.Now
	}
	// Resolved once so that a symlinked root is walked rather than seen as a single entry,
	// which would look like an empty library and empty the index.
	root, err := filepath.EvalSymlinks(s.Root)
	if err != nil {
		return Result{}, fmt.Errorf("library root: %w", err)
	}
	files, unreadable, err := walkLibrary(root)
	if err != nil {
		return Result{}, fmt.Errorf("library root: %w", err)
	}
	known, err := loadIndexed(ctx, s.DB)
	if err != nil {
		return Result{}, err
	}

	res := Result{Seen: len(files)}
	started := now()
	var todo []mediaFile
	for rel, f := range files {
		if k, ok := known[rel]; ok && !k.failed && k.size == f.size && k.mtimeNS == f.mtime.UnixNano() {
			continue
		}
		if s.settling(f.mtime, started) {
			res.Skipped++
			continue
		}
		todo = append(todo, f)
	}
	sort.Slice(todo, func(i, j int) bool { return todo[i].rel < todo[j].rel })

	// A block that has started is finished even if ctx is cancelled meanwhile.
	work := context.WithoutCancel(ctx)
	for i := 0; i < len(todo); i += blockSize {
		if err := ctx.Err(); err != nil {
			return res, err
		}
		block := todo[i:min(i+blockSize, len(todo))]
		failed, err := s.indexBlock(work, root, block, now())
		if err != nil {
			return res, err
		}
		res.Read += len(block)
		res.Errors += failed
		if progress != nil {
			progress(res.Read, len(todo))
		}
	}
	if err := ctx.Err(); err != nil {
		return res, err
	}
	res.Removed, err = removeVanished(ctx, s.DB, known, files, unreadable)
	return res, err
}

// IndexFiles reads the given files ('/'-separated, relative to Root) into the index right away,
// regardless of the quiet period, through the same block path as Run. Names that are not
// library media and entries that are not regular files are ignored.
func (s *Scanner) IndexFiles(ctx context.Context, rels []string) error {
	now := time.Now
	if s.Now != nil {
		now = s.Now
	}
	root, err := filepath.EvalSymlinks(s.Root)
	if err != nil {
		return fmt.Errorf("library root: %w", err)
	}
	var files []mediaFile
	for _, rel := range rels {
		kind, ok := mediaKinds[strings.ToLower(path.Ext(rel))]
		if !ok {
			continue
		}
		info, err := os.Lstat(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			return fmt.Errorf("index %s: %w", rel, err)
		}
		if !info.Mode().IsRegular() {
			continue
		}
		files = append(files, mediaFile{rel: rel, kind: kind, size: info.Size(), mtime: info.ModTime()})
	}
	for i := 0; i < len(files); i += blockSize {
		if _, err := s.indexBlock(ctx, root, files[i:min(i+blockSize, len(files))], now()); err != nil {
			return err
		}
	}
	return nil
}

// settling reports a file that may still be being written. An mtime further in the future
// than Quiet is a carried-over wrong timestamp, not a write in progress; treating it as
// settling would keep the file out of the index until that date.
func (s *Scanner) settling(mtime, now time.Time) bool {
	age := now.Sub(mtime)
	return s.Quiet > 0 && age < s.Quiet && age > -s.Quiet
}

func loadIndexed(ctx context.Context, db *sql.DB) (map[string]indexedFile, error) {
	rows, err := db.QueryContext(ctx, `SELECT rel_path, size, mtime_ns, read_error IS NOT NULL FROM photos`)
	if err != nil {
		return nil, fmt.Errorf("load index: %w", err)
	}
	defer rows.Close()
	known := map[string]indexedFile{}
	for rows.Next() {
		var rel string
		var k indexedFile
		if err := rows.Scan(&rel, &k.size, &k.mtimeNS, &k.failed); err != nil {
			return nil, fmt.Errorf("load index: %w", err)
		}
		known[rel] = k
	}
	return known, rows.Err()
}

// The upsert keeps a file's id stable across rescans; INSERT OR REPLACE would assign a new id
// and cascade-delete its tags.
const upsertPhoto = `INSERT INTO photos (rel_path, folder, media_type, format, size, mtime_ns,
	taken_at, taken_source, gps_lat, gps_lon, loc_sublocation, loc_city, loc_state, loc_country,
	loc_label, review_state, read_error, scanned_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(rel_path) DO UPDATE SET
	folder = excluded.folder, media_type = excluded.media_type, format = excluded.format,
	size = excluded.size, mtime_ns = excluded.mtime_ns, taken_at = excluded.taken_at,
	taken_source = excluded.taken_source, gps_lat = excluded.gps_lat, gps_lon = excluded.gps_lon,
	loc_sublocation = excluded.loc_sublocation, loc_city = excluded.loc_city,
	loc_state = excluded.loc_state, loc_country = excluded.loc_country,
	loc_label = excluded.loc_label, review_state = excluded.review_state,
	read_error = excluded.read_error, scanned_at = excluded.scanned_at
RETURNING id`

// indexBlock reads one block through ExifTool and writes it in one transaction. A file that
// cannot be read still gets a row (date from its name or mtime, no tags or place) so it is
// listed as unreadable and retried by every later scan.
func (s *Scanner) indexBlock(ctx context.Context, root string, block []mediaFile, at time.Time) (failed int, err error) {
	paths := make([]string, len(block))
	for i, f := range block {
		paths[i] = filepath.Join(root, filepath.FromSlash(f.rel))
	}
	recs, err := s.Reader.Read(ctx, paths)
	if err != nil {
		return 0, fmt.Errorf("read metadata: %w", err)
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	scannedAt := at.UTC().Format(time.RFC3339)
	for i, f := range block {
		rec, ok := recs[paths[i]]
		if !ok {
			rec = exif.Record{Err: "no result from ExifTool"}
		}
		m := metadata.Meta{ReviewState: "open"}
		var readErr any
		if rec.Err != "" {
			readErr = rec.Err
			failed++
		} else {
			m = metadata.Parse(rec.Fields)
		}
		takenAt, source := TakenDate(m, path.Base(f.rel), f.mtime, s.Loc)
		var id int64
		err := tx.QueryRowContext(ctx, upsertPhoto,
			f.rel, folderOf(f.rel), f.kind.mediaType, f.kind.format, f.size, f.mtime.UnixNano(),
			takenAt, source, floatOrNull(m.Lat), floatOrNull(m.Lon), orNull(m.Sublocation), orNull(m.City), orNull(m.State),
			orNull(m.Country), orNull(m.Label), m.ReviewState, readErr, scannedAt,
		).Scan(&id)
		if err != nil {
			return failed, fmt.Errorf("index %s: %w", f.rel, err)
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM photo_tags WHERE photo_id = ?`, id); err != nil {
			return failed, fmt.Errorf("index %s: %w", f.rel, err)
		}
		for _, tag := range m.Tags {
			if _, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO photo_tags (photo_id, tag_path) VALUES (?, ?)`, id, tag); err != nil {
				return failed, fmt.Errorf("index %s: %w", f.rel, err)
			}
		}
	}
	return failed, tx.Commit()
}

func orNull(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// removeVanished drops rows whose file was not seen. A row inside a folder that could not be
// listed stays: its file may well still exist. Tags go with the row (ON DELETE CASCADE).
func removeVanished(ctx context.Context, db *sql.DB, known map[string]indexedFile, files map[string]mediaFile, unreadable []string) (int, error) {
	var gone []string
	for rel := range known {
		if _, seen := files[rel]; seen || insideAny(rel, unreadable) {
			continue
		}
		gone = append(gone, rel)
	}
	if len(gone) == 0 {
		return 0, nil
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	for _, rel := range gone {
		if _, err := tx.ExecContext(ctx, `DELETE FROM photos WHERE rel_path = ?`, rel); err != nil {
			return 0, fmt.Errorf("remove %s: %w", rel, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return len(gone), nil
}

func insideAny(rel string, folders []string) bool {
	for _, f := range folders {
		if inside(rel, f) {
			return true
		}
	}
	return false
}

func floatOrNull(p *float64) any {
	if p == nil {
		return nil
	}
	return *p
}
