package ffmpeg

import (
	"fmt"
	"os/exec"
	"path/filepath"
)

// AudioProcessor handles audio extraction and processing
type AudioProcessor struct{}

// AudioTrack represents an audio track configuration
type AudioTrack struct {
	Language string
	Bitrate  string
	Channels int
}

func NewAudioProcessor() *AudioProcessor {
	return &AudioProcessor{}
}

// ExtractAudio extracts audio from video without re-encoding
func (p *AudioProcessor) ExtractAudio(inputPath, outputPath string) error {
	// Ensure output directory exists
	outputDir := filepath.Dir(outputPath)
	if err := ensureDir(outputDir); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	cmd := exec.Command("ffmpeg",
		"-i", inputPath,
		"-vn",
		"-acodec", "copy",
		"-y",
		outputPath,
	)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to extract audio: %w", err)
	}

	return nil
}

// ProcessAudio normalizes and processes audio
func (p *AudioProcessor) ProcessAudio(inputPath, outputPath string, bitrate string) error {
	// Normalize loudness + ensure stereo + encode to AAC
	cmd := exec.Command("ffmpeg",
		"-i", inputPath,
		"-af", "loudnorm=I=-16:TP=-1.5:LRA=11,silenceremove=1:0:-50dB",
		"-c:a", "aac",
		"-b:a", bitrate,
		"-ac", "2",
		"-y",
		outputPath,
	)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to process audio: %w", err)
	}

	return nil
}

// GenerateAudioHLS generates HLS segments for audio
func (p *AudioProcessor) GenerateAudioHLS(inputPath, outputDir, bitrate string, segmentDuration int) error {
	playlistPath := filepath.Join(outputDir, "main.m3u8")
	segmentPattern := filepath.Join(outputDir, "segment_%03d.aac")

	cmd := exec.Command("ffmpeg",
		"-i", inputPath,
		"-c:a", "aac",
		"-b:a", bitrate,
		"-ac", "2",
		"-threads", "0", // Use all CPU cores
		"-f", "hls",
		"-hls_time", fmt.Sprintf("%d", segmentDuration),
		"-hls_segment_type", "fmp4",
		"-hls_segment_filename", segmentPattern,
		"-hls_playlist_type", "vod",
		"-y",
		playlistPath,
	)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to generate audio HLS: %w", err)
	}

	return nil
}

// GenerateMultiQualityAudio generates multiple audio bitrates
func (p *AudioProcessor) GenerateMultiQualityAudio(inputPath, baseOutputDir string, segmentDuration int) ([]AudioQuality, error) {
	qualities := []struct {
		name    string
		bitrate string
	}{
		{"192k", "192k"},
		{"128k", "128k"},
		{"96k", "96k"},
	}

	var audioQualities []AudioQuality

	for _, q := range qualities {
		outputDir := filepath.Join(baseOutputDir, q.name)
		if err := ensureDir(outputDir); err != nil {
			return nil, err
		}

		if err := p.GenerateAudioHLS(inputPath, outputDir, q.bitrate, segmentDuration); err != nil {
			return nil, fmt.Errorf("failed to generate %s audio: %w", q.name, err)
		}

		audioQualities = append(audioQualities, AudioQuality{
			Bitrate:  q.bitrate,
			Playlist: filepath.Join(outputDir, "main.m3u8"),
		})
	}

	return audioQualities, nil
}

// AudioQuality represents an audio quality variant
type AudioQuality struct {
	Bitrate  string
	Playlist string
}

func ensureDir(path string) error {
	return exec.Command("mkdir", "-p", path).Run()
}
