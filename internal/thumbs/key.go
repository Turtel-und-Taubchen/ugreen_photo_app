// Package thumbs renders the WebP previews that live in the thumbnail cache.
package thumbs

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"strings"
)

// Key identifies one version of a file: a changed size or mtime yields a new key, so stale
// previews are never served.
func Key(relPath string, size, mtimeNs int64) string {
	sum := sha1.Sum([]byte(fmt.Sprintf("%s|%d|%d", relPath, size, mtimeNs)))
	return hex.EncodeToString(sum[:])
}

// Path is the cache file of a preview; size is "s" or "l".
func Path(cacheDir, key, size string) string {
	return base(cacheDir, key, size) + ".webp"
}

// FailedPath is the empty marker that records a failed render of the same preview.
func FailedPath(cacheDir, key, size string) string {
	return base(cacheDir, key, size) + ".failed"
}

func base(cacheDir, key, size string) string {
	return filepath.Join(cacheDir, key[:2], key+"-"+size)
}

// failedFor derives the marker path from a preview path.
func failedFor(dstWebp string) string {
	return strings.TrimSuffix(dstWebp, ".webp") + ".failed"
}
