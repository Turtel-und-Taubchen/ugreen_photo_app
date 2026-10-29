package probe

import (
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

const (
	scale  = 4
	width  = 640
	height = 480
	// glyphs per line on the small canvas before upscaling; longer markers wrap.
	charsPerLine = (width/scale - 8) / 7
)

// RenderText draws text black on white with the 7x13 bitmap font, then upscales by
// nearest neighbour so the glyphs stay crisp enough for OCR.
func RenderText(text string) *image.RGBA {
	small := image.NewRGBA(image.Rect(0, 0, width/scale, height/scale))
	draw.Draw(small, small.Bounds(), image.White, image.Point{}, draw.Src)
	d := &font.Drawer{Dst: small, Src: image.Black, Face: basicfont.Face7x13}
	y := 20
	for _, line := range wrapLines(text) {
		d.Dot = fixed.P(4, y)
		d.DrawString(line)
		y += 16
	}
	big := image.NewRGBA(image.Rect(0, 0, width, height))
	for yy := 0; yy < height; yy++ {
		for xx := 0; xx < width; xx++ {
			big.SetRGBA(xx, yy, small.RGBAAt(xx/scale, yy/scale))
		}
	}
	return big
}

func wrapLines(text string) []string {
	var out []string
	for _, line := range strings.Split(text, "\n") {
		r := []rune(line)
		for len(r) > charsPerLine {
			out = append(out, string(r[:charsPerLine]))
			r = r[charsPerLine:]
		}
		out = append(out, string(r))
	}
	return out
}

// WriteImage encodes img to path in the given format. HEIC goes through libvips from a
// temporary PNG because the Go standard library has no HEIF encoder.
func WriteImage(path string, f Format, img image.Image) error {
	switch f {
	case FormatJPEG:
		return encodeFile(path, func(w *os.File) error { return jpeg.Encode(w, img, &jpeg.Options{Quality: 95}) })
	case FormatPNG:
		return encodeFile(path, func(w *os.File) error { return png.Encode(w, img) })
	case FormatHEIC:
		tmp, err := os.MkdirTemp("", "pcprobe-heic-")
		if err != nil {
			return err
		}
		defer os.RemoveAll(tmp)
		src := filepath.Join(tmp, "in.png")
		if err := WriteImage(src, FormatPNG, img); err != nil {
			return err
		}
		if out, err := exec.Command("vips", "heifsave", src, path, "--Q", "90").CombinedOutput(); err != nil {
			return fmt.Errorf("vips heifsave: %w: %s", err, out)
		}
		return nil
	}
	return fmt.Errorf("no image encoder for format %q", f)
}

func encodeFile(path string, enc func(*os.File) error) error {
	w, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := enc(w); err != nil {
		w.Close()
		return err
	}
	return w.Close()
}
