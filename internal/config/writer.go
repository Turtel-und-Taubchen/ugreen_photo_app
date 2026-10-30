package config

import (
	"fmt"
	"strconv"
)

// Writer holds the settings of the metadata writer.
type Writer struct {
	PreserveMtime       bool
	BackupRetentionDays int
}

// WriterFromEnv reads PC_PRESERVE_MTIME (default true, only "true" or "false" are accepted)
// and PC_BACKUP_RETENTION_DAYS (default 30, a whole number of at least 1).
func WriterFromEnv(get func(string) string) (Writer, error) {
	w := Writer{PreserveMtime: true, BackupRetentionDays: 30}
	switch v := get("PC_PRESERVE_MTIME"); v {
	case "", "true":
	case "false":
		w.PreserveMtime = false
	default:
		return Writer{}, fmt.Errorf("PC_PRESERVE_MTIME=%q is not valid (use true or false)", v)
	}
	if v := get("PC_BACKUP_RETENTION_DAYS"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			return Writer{}, fmt.Errorf("PC_BACKUP_RETENTION_DAYS=%q is not valid (use a whole number of days, at least 1)", v)
		}
		w.BackupRetentionDays = n
	}
	return w, nil
}
