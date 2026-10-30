package config

import (
	"strings"
	"testing"
)

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
