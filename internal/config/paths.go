// Package config reads the container's path settings and validates them at startup.
package config

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"golang.org/x/sys/unix"
)

// Paths holds the library, state and backup locations.
type Paths struct {
	PhotosDir      string
	DataDir        string
	BackupDir      string
	PhotosHostPath string // optional; only checked for an Inbox segment
	UsersFile      string
}

// PathsFromEnv reads PC_PHOTOS_DIR, PC_DATA_DIR, PC_BACKUP_DIR and PC_PHOTOS_HOST_PATH, PC_USERS_FILE.
func PathsFromEnv(get func(string) string) Paths {
	def := func(key, d string) string {
		if v := get(key); v != "" {
			return v
		}
		return d
	}
	return Paths{
		PhotosDir:      def("PC_PHOTOS_DIR", "/photos"),
		DataDir:        def("PC_DATA_DIR", "/data"),
		BackupDir:      def("PC_BACKUP_DIR", "/backup"),
		PhotosHostPath: get("PC_PHOTOS_HOST_PATH"),
		UsersFile:      def("PC_USERS_FILE", "/data/users.conf"),
	}
}

// ValidatePaths reports every violation at once. Writability is decided by the kernel
// via access(2), never by evaluating mode bits or creating probe files.
func ValidatePaths(p Paths) error {
	var errs []error
	errs = append(errs, checkDir("photos", "PC_PHOTOS_DIR", p.PhotosDir)...)
	if hasInboxSegment(p.PhotosDir) {
		errs = append(errs, fmt.Errorf("photos path %q contains an Inbox segment: the library must not include the Inbox; mount the sorted library folder itself as PC_PHOTOS_DIR", p.PhotosDir))
	}
	if hasInboxSegment(p.PhotosHostPath) {
		errs = append(errs, fmt.Errorf("photos host path %q contains an Inbox segment: PC_PHOTOS_HOST_PATH must point at the sorted library, not the Inbox", p.PhotosHostPath))
	}
	if entries, err := os.ReadDir(p.PhotosDir); err == nil {
		for _, e := range entries {
			if e.IsDir() && strings.EqualFold(e.Name(), "inbox") {
				errs = append(errs, fmt.Errorf("photos directory %q directly contains the subfolder %q: remove or move the Inbox out of the library, or mount the sorted folder instead", p.PhotosDir, e.Name()))
			}
		}
	}
	errs = append(errs, checkDir("data", "PC_DATA_DIR", p.DataDir)...)
	errs = append(errs, checkDir("backup", "PC_BACKUP_DIR", p.BackupDir)...)
	return errors.Join(errs...)
}

func checkDir(label, env, path string) []error {
	fi, err := os.Stat(path)
	if err != nil {
		return []error{fmt.Errorf("%s path %q is not accessible (%v): create it or fix the mount for %s", label, path, err, env)}
	}
	if !fi.IsDir() {
		return []error{fmt.Errorf("%s path %q is not a directory: point %s at a directory", label, path, env)}
	}
	if err := unix.Access(path, unix.W_OK); err != nil {
		return []error{fmt.Errorf("%s path %q is not writable (%v): check that PUID/PGID may write to %q", label, path, err, path)}
	}
	return nil
}

func hasInboxSegment(path string) bool {
	segs := strings.FieldsFunc(path, func(r rune) bool { return r == '/' || r == '\\' })
	for _, s := range segs {
		if strings.EqualFold(strings.TrimSpace(s), "inbox") {
			return true
		}
	}
	return false
}
