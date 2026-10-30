package config

import (
	"fmt"
	"time"
)

// Scan holds the library scan cadence and the settle time for freshly changed files.
type Scan struct {
	Interval time.Duration
	Quiet    time.Duration
}

// ScanFromEnv reads PC_SCAN_INTERVAL (default 15m, at least 1m) and PC_SCAN_QUIET (default 5m, at least 0s).
func ScanFromEnv(get func(string) string) (Scan, error) {
	interval, err := durationEnv(get, "PC_SCAN_INTERVAL", 15*time.Minute, time.Minute)
	if err != nil {
		return Scan{}, err
	}
	quiet, err := durationEnv(get, "PC_SCAN_QUIET", 5*time.Minute, 0)
	if err != nil {
		return Scan{}, err
	}
	return Scan{Interval: interval, Quiet: quiet}, nil
}

func durationEnv(get func(string) string, key string, def, min time.Duration) (time.Duration, error) {
	v := get(key)
	if v == "" {
		return def, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("%s=%q is not a valid duration (e.g. 15m): %v", key, v, err)
	}
	if d < min {
		return 0, fmt.Errorf("%s=%q is below the minimum of %s", key, v, min)
	}
	return d, nil
}
