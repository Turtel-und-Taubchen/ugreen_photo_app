package config

import (
	"strings"
	"testing"
	"time"
)

func env(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestScanSettingsDefaultToA15MinuteIntervalAndA5MinuteQuietPeriodAndRejectInvalidValues(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		s, err := ScanFromEnv(env(nil))
		if err != nil || s.Interval != 15*time.Minute || s.Quiet != 5*time.Minute {
			t.Fatalf("got %+v, %v", s, err)
		}
	})
	t.Run("overrides", func(t *testing.T) {
		s, err := ScanFromEnv(env(map[string]string{"PC_SCAN_INTERVAL": "1m", "PC_SCAN_QUIET": "0s"}))
		if err != nil || s.Interval != time.Minute || s.Quiet != 0 {
			t.Fatalf("got %+v, %v", s, err)
		}
	})
	for _, tc := range []struct{ key, val string }{
		{"PC_SCAN_INTERVAL", "soon"},
		{"PC_SCAN_INTERVAL", "59s"},
		{"PC_SCAN_INTERVAL", "-5m"},
		{"PC_SCAN_QUIET", "5 minutes"},
		{"PC_SCAN_QUIET", "-1s"},
	} {
		t.Run(tc.key+"="+tc.val, func(t *testing.T) {
			_, err := ScanFromEnv(env(map[string]string{tc.key: tc.val}))
			if err == nil || !strings.Contains(err.Error(), tc.key) {
				t.Fatalf("want error naming %s, got %v", tc.key, err)
			}
		})
	}
}
