package probe

import (
	"fmt"
	"image"
	"os"
	"os/exec"
	"path/filepath"
)

// videoStamp is the QuickTime date of every video, written as UTC.
const videoStamp = "2001:01:01 11:00:00"

var videoDateTags = []string{"QuickTime:CreateDate", "QuickTime:ModifyDate", "QuickTime:TrackCreateDate", "QuickTime:MediaCreateDate"}

func generateVideos(outDir string) error {
	tmp, err := os.MkdirTemp("", "pcprobe-video-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	for _, p := range probes {
		if p.Format != FormatMP4 && p.Format != FormatMOV {
			continue
		}
		path := filepath.Join(outDir, p.File)
		if err := writeVideo(path, tmp, RenderText(p.PixelText())); err != nil {
			return fmt.Errorf("%s: %w", p.File, err)
		}
		assigns := []TagValue{}
		for _, t := range videoDateTags {
			assigns = append(assigns, TagValue{t, videoStamp})
		}
		files := []string{path}
		if p.Sidecar == "" {
			assigns = append(assigns, p.Fixed...)
			assigns = append(assigns, p.Texts...)
			assigns = append(assigns, p.GPS...)
		} else {
			sidecar := filepath.Join(outDir, p.Sidecar)
			// -o makes ExifTool create a standalone XMP file instead of editing the video.
			if err := writeTagsWith(path, []string{"-o", sidecar}, append(append([]TagValue{}, p.Texts...), p.GPS...)); err != nil {
				return err
			}
			files = append(files, sidecar)
		}
		if err := writeTagsWith(path, []string{"-overwrite_original", "-api", "QuickTimeUTC=1"}, assigns); err != nil {
			return err
		}
		for _, f := range files {
			if err := os.Chtimes(f, CaptureTime(), CaptureTime()); err != nil {
				return err
			}
		}
	}
	return nil
}

// writeVideo encodes a one-second clip of img without any container metadata.
func writeVideo(path, tmp string, img *image.RGBA) error {
	src := filepath.Join(tmp, "frame.png")
	if err := WriteImage(src, FormatPNG, img); err != nil {
		return err
	}
	out, err := exec.Command("ffmpeg", "-y", "-loop", "1", "-i", src, "-t", "1", "-r", "10",
		"-c:v", "libx264", "-pix_fmt", "yuv420p", "-map_metadata", "-1", path).CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg: %w: %s", err, out)
	}
	return nil
}
