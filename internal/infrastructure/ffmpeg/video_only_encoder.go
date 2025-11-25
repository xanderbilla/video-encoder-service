package ffmpeg

import (
	"fmt"
	"os/exec"
	"path/filepath"
)

// VideoOnlyEncoder encodes video without audio
type VideoOnlyEncoder struct{}

func NewVideoOnlyEncoder() *VideoOnlyEncoder {
	return &VideoOnlyEncoder{}
}

// EncodeVideoOnly encodes video track without audio
func (e *VideoOnlyEncoder) EncodeVideoOnly(inputPath, outputDir string, width, height int, bitrate string, segmentDuration int) error {
	playlistPath := filepath.Join(outputDir, "main.m3u8")
	segmentPattern := filepath.Join(outputDir, "segment_%03d.m4s")

	cmd := exec.Command("ffmpeg",
		"-i", inputPath,
		"-an", // No audio
		"-c:v", "libx264",
		"-preset", "superfast",
		"-crf", "23",
		"-threads", "0", // Use all CPU cores
		"-maxrate", bitrate,
		"-bufsize", fmt.Sprintf("%dk", parseInt(bitrate)*2),
		"-vf", fmt.Sprintf("scale=%d:%d", width, height),
		"-g", "48",
		"-keyint_min", "48",
		"-sc_threshold", "0",
		"-f", "hls",
		"-hls_time", fmt.Sprintf("%d", segmentDuration),
		"-hls_segment_type", "fmp4",
		"-hls_segment_filename", segmentPattern,
		"-hls_playlist_type", "vod",
		"-y",
		playlistPath,
	)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to encode video-only: %w", err)
	}

	return nil
}

func parseInt(bitrate string) int {
	// Parse bitrate string like "5000k" to int
	var val int
	fmt.Sscanf(bitrate, "%dk", &val)
	if val == 0 {
		val = 2500 // default
	}
	return val
}
