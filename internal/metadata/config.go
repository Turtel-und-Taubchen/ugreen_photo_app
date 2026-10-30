// Package metadata holds the app's own ExifTool vocabulary.
package metadata

import (
	_ "embed"
	"os"
	"path/filepath"
)

//go:embed photocurator.config
var configFile []byte

// WriteConfig writes the ExifTool configuration into dir and returns its path. ExifTool can
// only read -config from a real file, and the container's root filesystem is read-only, so
// callers pass a directory on tmpfs (os.TempDir()).
func WriteConfig(dir string) (path string, err error) {
	path = filepath.Join(dir, "photocurator.config")
	if err := os.WriteFile(path, configFile, 0o644); err != nil {
		return "", err
	}
	return path, nil
}
