package writer

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// inflightDir holds the pre-copy of every file whose write has started but not finished:
// <BackupDir>/.inflight/<YYYY-MM-DD>/<rel>. A copy that is still there means the original may
// be half-written and must be restored from it.
const inflightDir = ".inflight"

// Partial pre-copies carry this prefix and the tmpSuffix, which no library format uses, so
// Recover can tell them from real pre-copies and drop them.
const (
	tmpPrefix = ".pc-inflight-"
	tmpSuffix = ".tmp"
)

func (p *Pipeline) inflightRoot() string { return filepath.Join(p.BackupDir, inflightDir) }

// pendingCopy returns an existing pre-copy of rel under any date, or "".
func (p *Pipeline) pendingCopy(rel string) (string, error) {
	days, err := os.ReadDir(p.inflightRoot())
	if errors.Is(err, fs.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	for _, d := range days {
		c := filepath.Join(p.inflightRoot(), d.Name(), filepath.FromSlash(rel))
		if _, err := os.Lstat(c); err == nil {
			return c, nil
		} else if !errors.Is(err, fs.ErrNotExist) {
			return "", err
		}
	}
	return "", nil
}

// preCopy writes src's bytes to dst durably: a temp file in dst's folder is synced, given
// src's mtime and renamed, so dst is either absent or complete.
func preCopy(dst, src string, mtime time.Time) (err error) {
	dir := filepath.Dir(dst)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	tmp, err := os.CreateTemp(dir, tmpPrefix+"*"+tmpSuffix)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			// Best effort: err is the failure to report; Recover drops a leftover temp file.
			os.Remove(tmp.Name())
		}
	}()
	_, err = io.Copy(tmp, in)
	if err == nil {
		err = tmp.Sync()
	}
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return err
	}
	if err := os.Chtimes(tmp.Name(), time.Time{}, mtime); err != nil {
		return err
	}
	if err := os.Rename(tmp.Name(), dst); err != nil {
		return err
	}
	return syncDir(dir)
}

func syncDir(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
}

// restoreInPlace overwrites the original through its existing inode, so owner, mode and hard
// links stay as they were. The original is never created: a missing one is an error.
func restoreInPlace(abs, copyPath string) error {
	src, err := os.Open(copyPath)
	if err != nil {
		return err
	}
	defer src.Close()
	info, err := src.Stat()
	if err != nil {
		return err
	}
	dst, err := os.OpenFile(abs, os.O_WRONLY|os.O_TRUNC, 0)
	if err != nil {
		return err
	}
	_, err = io.Copy(dst, src)
	if err == nil {
		err = dst.Sync()
	}
	if cerr := dst.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return err
	}
	return os.Chtimes(abs, time.Time{}, info.ModTime())
}

// sameBytes reports whether both files hold identical bytes. An original that still equals its
// pre-copy needs no restore, which matters when it cannot be opened for writing.
func sameBytes(a, b string) (bool, error) {
	fa, err := os.Open(a)
	if err != nil {
		return false, err
	}
	defer fa.Close()
	fb, err := os.Open(b)
	if err != nil {
		return false, err
	}
	defer fb.Close()
	ia, err := fa.Stat()
	if err != nil {
		return false, err
	}
	ib, err := fb.Stat()
	if err != nil {
		return false, err
	}
	if ia.Size() != ib.Size() {
		return false, nil
	}
	bufA, bufB := make([]byte, 64<<10), make([]byte, 64<<10)
	for {
		na, errA := io.ReadFull(fa, bufA)
		nb, errB := io.ReadFull(fb, bufB)
		if na != nb || !bytes.Equal(bufA[:na], bufB[:nb]) {
			return false, nil
		}
		if errA != nil || errB != nil {
			for _, e := range []error{errA, errB} {
				if e != nil && !errors.Is(e, io.EOF) && !errors.Is(e, io.ErrUnexpectedEOF) {
					return false, e
				}
			}
			return true, nil
		}
	}
}

// dropCopy deletes a pre-copy and the folders that became empty, up to .inflight itself.
func (p *Pipeline) dropCopy(copyPath string) error {
	if err := os.Remove(copyPath); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	stop := p.inflightRoot()
	for dir := filepath.Dir(copyPath); dir != stop && strings.HasPrefix(dir, stop+string(filepath.Separator)); dir = filepath.Dir(dir) {
		// Removal fails while the folder still holds other copies; that is the stop signal, not an error.
		if os.Remove(dir) != nil {
			break
		}
	}
	return nil
}

// Recover writes every pre-copy left by an interrupted write back into its original, in place,
// deletes the copy and re-indexes the file. It returns the restored files ('/'-separated). A
// copy whose original cannot be restored stays for the next attempt and is reported in the
// error; the other files are still recovered.
func (p *Pipeline) Recover(ctx context.Context) ([]string, error) {
	root := p.inflightRoot()
	type pending struct{ rel, copyPath string }
	var todo []pending
	var errs []error
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			if path == root && errors.Is(walkErr, fs.ErrNotExist) {
				return filepath.SkipAll
			}
			errs = append(errs, walkErr)
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasPrefix(d.Name(), tmpPrefix) && strings.HasSuffix(d.Name(), tmpSuffix) {
			if err := os.Remove(path); err != nil {
				errs = append(errs, err)
			}
			return nil
		}
		r, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		_, rel, ok := strings.Cut(filepath.ToSlash(r), "/")
		if !ok {
			errs = append(errs, fmt.Errorf("in-flight copy %s outside a date folder", r))
			return nil
		}
		todo = append(todo, pending{rel, path})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(todo, func(i, j int) bool { return todo[i].rel < todo[j].rel })

	var restored []string
	for _, t := range todo {
		abs, _, _, reason := p.resolve(t.rel)
		if reason != "" {
			errs = append(errs, fmt.Errorf("recover %s: %s", t.rel, reason))
			continue
		}
		if same, err := sameBytes(abs, t.copyPath); err == nil && same {
			// The write never changed the original, so there is nothing to restore or re-index.
			if err := p.dropCopy(t.copyPath); err != nil {
				errs = append(errs, fmt.Errorf("recover %s: %w", t.rel, err))
			}
			continue
		}
		if err := restoreInPlace(abs, t.copyPath); err != nil {
			errs = append(errs, fmt.Errorf("recover %s: %w", t.rel, err))
			continue
		}
		if err := p.dropCopy(t.copyPath); err != nil {
			errs = append(errs, fmt.Errorf("recover %s: %w", t.rel, err))
		}
		restored = append(restored, t.rel)
	}
	if len(restored) > 0 && p.Index != nil {
		if err := p.Index.IndexFiles(context.WithoutCancel(ctx), restored); err != nil {
			errs = append(errs, err)
		}
	}
	return restored, errors.Join(errs...)
}
