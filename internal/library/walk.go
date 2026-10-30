package library

import (
	"errors"
	"io/fs"
	"path"
	"path/filepath"
	"strings"
	"time"
)

type mediaKind struct{ format, mediaType string }

// mediaKinds maps a lower-case extension to the indexed format and media type.
var mediaKinds = map[string]mediaKind{
	".jpg": {"jpeg", "image"}, ".jpeg": {"jpeg", "image"},
	".heic": {"heic", "image"}, ".heif": {"heic", "image"},
	".png": {"png", "image"},
	".mp4": {"mp4", "video"},
	".mov": {"mov", "video"},
}

type mediaFile struct {
	rel   string // '/'-separated, relative to the root
	kind  mediaKind
	size  int64
	mtime time.Time
}

// ignoredName matches hidden entries, NAS system folders such as @eaDir and the recycle bin.
func ignoredName(name string) bool {
	return strings.HasPrefix(name, ".") || strings.HasPrefix(name, "@") || name == "#recycle"
}

// folderOf is the parent folder of rel; files directly in the root have folder "".
func folderOf(rel string) string {
	if d := path.Dir(rel); d != "." {
		return d
	}
	return ""
}

// inside reports whether rel lies below folder ("" is the root and contains everything).
func inside(rel, folder string) bool {
	return folder == "" || strings.HasPrefix(rel, folder+"/")
}

// walkLibrary lists the media files below root without following symlinks and without
// opening any file. unreadable holds the folders whose listing was incomplete: a file missing
// from files but inside one of them may still exist. An unreadable root is an error, because
// then nothing at all was seen.
func walkLibrary(root string) (files map[string]mediaFile, unreadable []string, err error) {
	files = map[string]mediaFile{}
	err = filepath.WalkDir(root, func(p string, d fs.DirEntry, walkErr error) error {
		if p == root {
			return walkErr
		}
		r, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		rel := filepath.ToSlash(r)
		if walkErr != nil {
			// Only a failed directory listing reaches here; WalkDir still visits whatever part
			// of the listing it got.
			unreadable = append(unreadable, rel)
			return nil
		}
		if ignoredName(d.Name()) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		kind, ok := mediaKinds[strings.ToLower(filepath.Ext(d.Name()))]
		if !ok {
			return nil
		}
		info, err := d.Info() // Lstat
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		if err != nil {
			unreadable = append(unreadable, folderOf(rel))
			return nil
		}
		files[rel] = mediaFile{rel: rel, kind: kind, size: info.Size(), mtime: info.ModTime()}
		return nil
	})
	return files, unreadable, err
}
