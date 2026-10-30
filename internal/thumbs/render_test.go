package thumbs

import (
	"context"
	"crypto/sha256"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"photocurator/internal/testsupport"
)

type env struct {
	lib, cache string
	r          Renderer
}

func newEnv(t *testing.T) env {
	t.Helper()
	lib := testsupport.CopyFixture(t, "library")
	cache := t.TempDir()
	return env{lib: lib, cache: cache, r: Renderer{CacheDir: cache, Root: lib}}
}

func (e env) abs(rel string) string { return filepath.Join(e.lib, filepath.FromSlash(rel)) }

// render renders rel in the given size and returns the preview path.
func (e env) render(t *testing.T, rel, mediaType, size string) (string, error) {
	t.Helper()
	dst := Path(e.cache, Key(rel, 1, 1), size)
	return dst, e.r.RenderTo(context.Background(), e.abs(rel), mediaType, size, dst)
}

func out(t *testing.T, name string, args ...string) string {
	t.Helper()
	b, err := exec.Command(name, args...).CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v: %v: %s", name, args, err, b)
	}
	return strings.TrimSpace(string(b))
}

func dims(t *testing.T, path string) (w, h int) {
	t.Helper()
	w, _ = strconv.Atoi(out(t, "vipsheader", "-f", "width", path))
	h, _ = strconv.Atoi(out(t, "vipsheader", "-f", "height", path))
	return
}

// centerRGB is the pixel in the middle of the image.
func centerRGB(t *testing.T, path string) (r, g, b float64) {
	t.Helper()
	w, h := dims(t, path)
	f := strings.Fields(out(t, "vips", "getpoint", path, strconv.Itoa(w/2), strconv.Itoa(h/2)))
	if len(f) < 3 {
		t.Fatalf("getpoint: %v", f)
	}
	r, _ = strconv.ParseFloat(f[0], 64)
	g, _ = strconv.ParseFloat(f[1], 64)
	b, _ = strconv.ParseFloat(f[2], 64)
	return
}

func isWebP(t *testing.T, path string) {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil || len(b) < 12 || string(b[:4]) != "RIFF" || string(b[8:12]) != "WEBP" {
		t.Fatalf("%s is not a WebP file (err %v)", path, err)
	}
}

func TestImagesBecome320And1600PxWebpPreviewsWithoutUpscaling(t *testing.T) {
	e := newEnv(t)
	for _, rel := range []string{"2024/07/sample.jpg", "2024/08/screenshot.png", "2024/07/sample.heic"} {
		sw, sh := dims(t, e.abs(rel))
		for size, limit := range map[string]int{"s": 320, "l": 1600} {
			dst, err := e.render(t, rel, "image", size)
			if err != nil {
				t.Fatalf("%s %s: %v", rel, size, err)
			}
			isWebP(t, dst)
			w, h := dims(t, dst)
			if max(w, h) > limit || w > sw || h > sh {
				t.Errorf("%s %s: %dx%d from %dx%d, limit %d", rel, size, w, h, sw, sh, limit)
			}
		}
	}
}

func TestAnExifRotatedJpegYieldsAPortraitThumbnail(t *testing.T) {
	e := newEnv(t)
	rel := "2024/07/sample.jpg"
	if w, h := dims(t, e.abs(rel)); w <= h {
		t.Fatalf("fixture must be landscape, is %dx%d", w, h)
	}
	out(t, "exiftool", "-overwrite_original", "-n", "-Orientation=6", e.abs(rel))
	dst, err := e.render(t, rel, "image", "s")
	if err != nil {
		t.Fatal(err)
	}
	if w, h := dims(t, dst); w != 240 || h != 320 {
		t.Errorf("got %dx%d, want 240x320", w, h)
	}
}

func TestAVideoPreviewIsTheFrameAtOneSecond(t *testing.T) {
	e := newEnv(t)
	dst, err := e.render(t, "2024/08/clip.mp4", "video", "s")
	if err != nil {
		t.Fatal(err)
	}
	if r, g, b := centerRGB(t, dst); b < 150 || r > 100 || g > 100 {
		t.Errorf("center pixel %v,%v,%v is not blue", r, g, b)
	}
}

func TestAVideoShorterThanOneSecondUsesItsFirstFrame(t *testing.T) {
	e := newEnv(t)
	dst, err := e.render(t, "2024/08/short.mov", "video", "s")
	if err != nil {
		t.Fatal(err)
	}
	isWebP(t, dst)
	if r, g, b := centerRGB(t, dst); r < 150 || g > 100 || b > 100 {
		t.Errorf("center pixel %v,%v,%v is not red", r, g, b)
	}
}

func TestABrokenFileFailsAndLeavesAFailureMarker(t *testing.T) {
	e := newEnv(t)
	key := Key("2024/08/defekt.jpg", 1, 1)
	if _, err := e.render(t, "2024/08/defekt.jpg", "image", "s"); err == nil {
		t.Fatal("expected an error")
	}
	if st, err := os.Stat(FailedPath(e.cache, key, "s")); err != nil || st.Size() != 0 {
		t.Errorf("failure marker: %v %v", st, err)
	}
	if _, err := os.Stat(Path(e.cache, key, "s")); err == nil {
		t.Error("a preview exists for a broken file")
	}
}

func TestRendererProcessesRunAtNice10(t *testing.T) {
	ctx := context.Background()
	for _, cmd := range []*exec.Cmd{
		vipsCmd(ctx, "/in.jpg", "320x320>", "/out.webp"),
		ffmpegCmd(ctx, "/in.mp4", true, "/out.png"),
		ffmpegCmd(ctx, "/in.mp4", false, "/out.png"),
	} {
		if len(cmd.Args) < 4 || !reflect.DeepEqual(cmd.Args[:3], []string{"nice", "-n", "10"}) {
			t.Errorf("command does not start with nice -n 10: %v", cmd.Args)
		}
	}
}

type fileInfo struct {
	size int64
	mt   int64
	sum  [32]byte
}

func snapshot(t *testing.T, root string) map[string]fileInfo {
	t.Helper()
	m := map[string]fileInfo{}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		st, err := d.Info()
		if err != nil {
			return err
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, p)
		m[rel] = fileInfo{st.Size(), st.ModTime().UnixNano(), sha256.Sum256(b)}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestRenderingLeavesTheLibraryUntouchedAndWritesOnlyIntoTheCache(t *testing.T) {
	e := newEnv(t)
	before := snapshot(t, e.lib)
	for rel := range before {
		mt := "image"
		if strings.HasSuffix(rel, ".mp4") || strings.HasSuffix(rel, ".mov") {
			mt = "video"
		}
		for _, size := range []string{"s", "l"} {
			e.render(t, filepath.ToSlash(rel), mt, size) // notizen.txt and defekt.jpg are expected to fail
		}
	}
	if after := snapshot(t, e.lib); !reflect.DeepEqual(before, after) {
		t.Errorf("library changed:\nbefore %v\nafter  %v", before, after)
	}
	cached := snapshot(t, e.cache)
	if len(cached) == 0 {
		t.Fatal("nothing was rendered into the cache")
	}
	for rel := range cached {
		if ext := filepath.Ext(rel); ext != ".webp" && ext != ".failed" {
			t.Errorf("unexpected cache file %s", rel)
		}
	}
}
