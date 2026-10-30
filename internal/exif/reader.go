// Package exif reads metadata through ExifTool.
package exif

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Reader reads a targeted tag list from batches of files. Read-only: no option that writes,
// copies or assigns is ever passed.
type Reader struct {
	ConfigPath string   // ExifTool -config file (see metadata.WriteConfig)
	Tags       []string // tags to read, "Tag" or "Group:Tag"; empty reads everything
}

// Record is the result for one requested file. Fields are keyed "Group:Tag" (family 1).
type Record struct {
	Fields map[string]any
	Err    string
}

// Read runs one ExifTool process for all paths. Results are matched to paths by the
// SourceFile ExifTool echoes, never by position: an unreadable file may produce no JSON
// object at all, which would shift every later one. Only a failure to run ExifTool or to
// parse its output is an error; a non-zero exit with partial results is normal.
func (r Reader) Read(ctx context.Context, paths []string) (map[string]Record, error) {
	out := make(map[string]Record, len(paths))
	var abs []string // path as passed to ExifTool -> request key
	keyOf := make(map[string]string, len(paths))
	for _, p := range paths {
		// The argument file is line-based, so a line break would smuggle in an option.
		if strings.ContainsAny(p, "\r\n") {
			out[p] = Record{Err: "unsupported file name"}
			continue
		}
		a, err := filepath.Abs(p)
		if err != nil {
			out[p] = Record{Err: "not readable"}
			continue
		}
		if _, dup := keyOf[a]; dup {
			continue
		}
		keyOf[a] = p
		abs = append(abs, a)
	}
	if len(abs) == 0 {
		return out, nil
	}

	var b strings.Builder
	b.WriteString("-j\n-n\n-G1\n-a\n")
	for _, t := range r.Tags {
		if strings.ContainsAny(t, "\r\n") {
			return nil, fmt.Errorf("line break in tag %q", t)
		}
		b.WriteString("-" + t + "\n")
	}
	// No -fast2: it stops before the moov atom of videos that carry it behind mdat and
	// silently drops QuickTime dates and GPS.
	b.WriteString("--\n")
	for _, a := range abs {
		b.WriteString(a + "\n")
	}
	f, err := os.CreateTemp("", "pcexif-args-*.txt")
	if err != nil {
		return nil, err
	}
	defer os.Remove(f.Name())
	if _, err := f.WriteString(b.String()); err != nil {
		f.Close()
		return nil, err
	}
	if err := f.Close(); err != nil {
		return nil, err
	}

	args := []string{}
	if r.ConfigPath != "" {
		args = append(args, "-config", r.ConfigPath)
	}
	args = append(args, "-@", f.Name())
	cmd := exec.CommandContext(ctx, "exiftool", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	runErr := cmd.Run()
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	var objs []map[string]any
	if stdout.Len() > 0 {
		if err := json.Unmarshal(stdout.Bytes(), &objs); err != nil {
			return nil, fmt.Errorf("exiftool json: %w", err)
		}
	} else if runErr != nil {
		var ee *exec.ExitError
		if !errors.As(runErr, &ee) {
			return nil, fmt.Errorf("exiftool: %w", runErr)
		}
	}

	for _, o := range objs {
		src, _ := o["SourceFile"].(string)
		key, ok := keyOf[src]
		if !ok {
			continue
		}
		rec := Record{Fields: make(map[string]any, len(o))}
		for k, v := range o {
			if k == "SourceFile" {
				continue
			}
			if k == "Error" || k == "ExifTool:Error" {
				rec.Err = fmt.Sprint(v)
				continue
			}
			rec.Fields[k] = v
		}
		out[key] = rec
	}

	// ExifTool reports a file it cannot read only on stderr, and then emits either no object
	// or an object holding nothing but SourceFile.
	lines := strings.Split(stderr.String(), "\n")
	for _, a := range abs {
		key := keyOf[a]
		rec, ok := out[key]
		if ok && (rec.Err != "" || len(rec.Fields) > 0) {
			continue
		}
		msg := stderrFor(lines, a)
		if msg == "" && !ok {
			msg = "not readable"
		}
		if msg != "" {
			out[key] = Record{Err: msg}
		}
	}
	return out, nil
}

// stderrFor picks the ExifTool message that names path, so one bad file's text is never
// attributed to another.
func stderrFor(lines []string, path string) string {
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l != "" && strings.Contains(l, path) {
			return strings.TrimPrefix(l, "Error: ")
		}
	}
	return ""
}
