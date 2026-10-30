package thumbs

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const renderTimeout = 120 * time.Second

var edges = map[string]string{"s": "320x320>", "l": "1600x1600>"}

// Renderer writes previews below CacheDir and nowhere else. Root is only used to log
// library-relative paths.
type Renderer struct {
	CacheDir string
	Root     string
}

// RenderTo renders absSrc to dstWebp (which must lie below CacheDir) via a temp file in the
// same folder, so readers never see a partial preview. On failure it leaves an empty
// .failed marker next to dstWebp and returns the error.
func (r Renderer) RenderTo(ctx context.Context, absSrc, mediaType, size, dstWebp string) error {
	err := r.render(ctx, absSrc, mediaType, size, dstWebp)
	if err != nil {
		slog.Warn("thumbnail failed", "path", r.rel(absSrc), "size", size, "err", err)
		if werr := writeMarker(failedFor(dstWebp)); werr != nil {
			slog.Warn("thumbnail failure marker", "path", r.rel(absSrc), "err", werr)
		}
	}
	return err
}

func (r Renderer) render(ctx context.Context, absSrc, mediaType, size, dstWebp string) error {
	edge, ok := edges[size]
	if !ok {
		return fmt.Errorf("unknown size %q", size)
	}
	ctx, cancel := context.WithTimeout(ctx, renderTimeout)
	defer cancel()

	src := absSrc
	if mediaType == "video" {
		frame, err := r.extractFrame(ctx, absSrc)
		if err != nil {
			return err
		}
		defer os.Remove(frame)
		src = frame
	}
	dir := filepath.Dir(dstWebp)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := tempFile(dir, ".tmp-*.webp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp)
	if err := run(vipsCmd(ctx, src, edge, tmp)); err != nil {
		return err
	}
	if !nonEmpty(tmp) {
		return errors.New("vipsthumbnail produced no output")
	}
	return os.Rename(tmp, dstWebp)
}

// extractFrame grabs the frame at one second; videos shorter than that yield no frame, so
// the retry takes the first one.
func (r Renderer) extractFrame(ctx context.Context, absSrc string) (string, error) {
	tmpDir := filepath.Join(r.CacheDir, "tmp")
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		return "", err
	}
	frame, err := tempFile(tmpDir, "*.png")
	if err != nil {
		return "", err
	}
	for _, seek := range []bool{true, false} {
		if err := run(ffmpegCmd(ctx, absSrc, seek, frame)); err == nil && nonEmpty(frame) {
			return frame, nil
		}
		if ctx.Err() != nil {
			break
		}
	}
	os.Remove(frame)
	return "", errors.New("ffmpeg produced no frame")
}

// vipsCmd's output path must be absolute, otherwise vipsthumbnail writes next to the source.
func vipsCmd(ctx context.Context, src, edge, outAbs string) *exec.Cmd {
	return command(ctx, "vipsthumbnail", src, "--size", edge, "--export-profile", "srgb",
		"-o", outAbs+"[Q=75,strip]")
}

func ffmpegCmd(ctx context.Context, src string, seek bool, outPNG string) *exec.Cmd {
	args := []string{"-nostdin", "-y", "-loglevel", "error"}
	if seek {
		args = append(args, "-ss", "1")
	}
	args = append(args, "-i", src, "-frames:v", "1", outPNG)
	return command(ctx, "ffmpeg", args...)
}

// command prefixes every renderer process with nice: previews must not compete with the
// web server for CPU.
func command(ctx context.Context, name string, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, "nice", append([]string{"-n", "10", name}, args...)...)
}

func run(cmd *exec.Cmd) error {
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %w: %s", cmd.Args[3], err, strings.TrimSpace(string(out)))
	}
	return nil
}

func tempFile(dir, pattern string) (string, error) {
	f, err := os.CreateTemp(dir, pattern)
	if err != nil {
		return "", err
	}
	f.Close()
	return f.Name(), nil
}

func nonEmpty(path string) bool {
	st, err := os.Stat(path)
	return err == nil && st.Size() > 0
}

func writeMarker(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	return f.Close()
}

func (r Renderer) rel(abs string) string {
	if rel, err := filepath.Rel(r.Root, abs); r.Root != "" && err == nil && !strings.HasPrefix(rel, "..") {
		return filepath.ToSlash(rel)
	}
	return filepath.Base(abs)
}
