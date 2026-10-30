package config

import (
	"strings"
	"testing"
)

func TestBackupRetentionDefaultsTo30DaysAndRejectsInvalidValues(t *testing.T) {
	for val, want := range map[string]int{"": 30, "1": 1, "90": 90} {
		w, err := WriterFromEnv(env(map[string]string{"PC_BACKUP_RETENTION_DAYS": val}))
		if err != nil || w.BackupRetentionDays != want {
			t.Fatalf("%q: got %+v, %v", val, w, err)
		}
	}
	for _, val := range []string{"0", "-3", "abc", "1.5", "30d"} {
		_, err := WriterFromEnv(env(map[string]string{"PC_BACKUP_RETENTION_DAYS": val}))
		if err == nil || !strings.Contains(err.Error(), "PC_BACKUP_RETENTION_DAYS") {
			t.Fatalf("%q: want error naming PC_BACKUP_RETENTION_DAYS, got %v", val, err)
		}
	}
}

func TestPreserveMtimeDefaultsToTrueAndRejectsInvalidValues(t *testing.T) {
	for val, want := range map[string]bool{"": true, "true": true, "false": false} {
		w, err := WriterFromEnv(env(map[string]string{"PC_PRESERVE_MTIME": val}))
		if err != nil || w.PreserveMtime != want {
			t.Fatalf("%q: got %+v, %v", val, w, err)
		}
	}
	for _, val := range []string{"yes", "1", "TRUE", "False", " true"} {
		_, err := WriterFromEnv(env(map[string]string{"PC_PRESERVE_MTIME": val}))
		if err == nil || !strings.Contains(err.Error(), "PC_PRESERVE_MTIME") {
			t.Fatalf("%q: want error naming PC_PRESERVE_MTIME, got %v", val, err)
		}
	}
}
