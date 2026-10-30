package exif

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

const (
	readyMarker    = "{ready}"
	readyErrMarker = "{ready-err}"
)

// Writer changes metadata through one long-lived ExifTool process (-stay_open), so a job
// over many files pays the start-up cost once. Calls are serialised: the stdin/stdout
// protocol carries exactly one command at a time. The process is started on first use and
// restarted after it died or a call was cancelled.
type Writer struct {
	ConfigPath    string // ExifTool -config file (see metadata.WriteConfig)
	PreserveMtime bool   // pass -P so the file's modification time stays untouched

	mu     sync.Mutex
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Reader
	stderr *bufio.Reader
}

// Write runs ExifTool with args on path. In-place overwrite is always added: it keeps inode,
// owner and permissions and leaves no _original file. A reported "Error:" is returned.
func (w *Writer) Write(ctx context.Context, path string, args []string) error {
	cmdArgs := append([]string{}, args...)
	cmdArgs = append(cmdArgs, "-overwrite_original_in_place")
	if w.PreserveMtime {
		cmdArgs = append(cmdArgs, "-P")
	}
	_, err := w.run(ctx, path, cmdArgs)
	return err
}

// ImageHash returns the SHA-256 over the image data only, which metadata edits must not change.
func (w *Writer) ImageHash(ctx context.Context, path string) (string, error) {
	out, err := w.run(ctx, path, []string{"-api", "ImageHashType=SHA256", "-ImageDataHash", "-s3", "-f"})
	if err != nil {
		return "", err
	}
	h := strings.TrimSpace(out)
	if h == "" || h == "-" {
		return "", errors.New("no image data hash")
	}
	return h, nil
}

// Close ends the ExifTool process. The Writer can be used again afterwards.
func (w *Writer) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.stop(true)
}

// stop ends the process; graceful asks ExifTool to exit itself first.
func (w *Writer) stop(graceful bool) error {
	if w.cmd == nil {
		return nil
	}
	// Errors of the three calls below are ignored on purpose: the process may already be gone,
	// and Wait reports whether it ended cleanly.
	if graceful {
		io.WriteString(w.stdin, "-stay_open\nFalse\n")
	}
	w.stdin.Close()
	if !graceful {
		w.cmd.Process.Kill()
	}
	err := w.cmd.Wait()
	w.cmd, w.stdin, w.stdout, w.stderr = nil, nil, nil, nil
	if graceful && err == nil {
		return nil
	}
	if graceful {
		return err
	}
	return nil
}

func (w *Writer) start() error {
	args := []string{}
	if w.ConfigPath != "" {
		// -config is only honoured as the first argument of the process.
		args = append(args, "-config", w.ConfigPath)
	}
	args = append(args, "-stay_open", "True", "-@", "-")
	cmd := exec.Command("exiftool", args...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("exiftool: %w", err)
	}
	w.cmd, w.stdin = cmd, stdin
	w.stdout, w.stderr = bufio.NewReader(stdout), bufio.NewReader(stderr)
	return nil
}

func (w *Writer) run(ctx context.Context, path string, args []string) (string, error) {
	// The command stream is line-based, so a line break would smuggle in an option.
	for _, a := range append([]string{path}, args...) {
		if strings.ContainsAny(a, "\r\n") {
			return "", errors.New("line break in exiftool argument")
		}
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	w.mu.Lock()
	defer w.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if w.cmd == nil {
		if err := w.start(); err != nil {
			return "", err
		}
	}

	// A cancelled call leaves the protocol mid-command, so the process is killed; the next
	// call starts a fresh one.
	done := make(chan struct{})
	killed := make(chan struct{})
	go func() {
		defer close(killed)
		select {
		case <-ctx.Done():
			w.cmd.Process.Kill()
		case <-done:
		}
	}()
	out, errText, runErr := w.exchange(args, abs)
	close(done)
	<-killed
	if runErr != nil {
		w.stop(false)
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return "", runErr
	}
	if errText != "" {
		return out, errors.New(errText)
	}
	return out, nil
}

// exchange sends one command and reads until both streams reached their end marker.
func (w *Writer) exchange(args []string, abs string) (out, errText string, err error) {
	var b strings.Builder
	for _, a := range args {
		b.WriteString(a + "\n")
	}
	b.WriteString("-echo4\n" + readyErrMarker + "\n" + abs + "\n-execute\n")
	if _, err := io.WriteString(w.stdin, b.String()); err != nil {
		return "", "", fmt.Errorf("exiftool: %w", err)
	}

	var ob strings.Builder
	for {
		line, err := w.stdout.ReadString('\n')
		if err != nil {
			return "", "", fmt.Errorf("exiftool ended: %w", err)
		}
		if strings.TrimSpace(line) == readyMarker {
			break
		}
		ob.WriteString(line)
	}
	var msgs []string
	for {
		line, err := w.stderr.ReadString('\n')
		if err != nil {
			return "", "", fmt.Errorf("exiftool ended: %w", err)
		}
		line = strings.TrimSpace(line)
		if line == readyErrMarker {
			break
		}
		if strings.HasPrefix(line, "Error:") {
			msgs = append(msgs, strings.TrimSpace(strings.TrimPrefix(line, "Error:")))
		}
	}
	return ob.String(), strings.Join(msgs, "; "), nil
}
