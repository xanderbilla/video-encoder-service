package ffmpeg

import (
	"encoder-service/pkg/config"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// PreviewGenerator handles thumbnail and GIF generation
type PreviewGenerator struct {
	config *config.Config
}

// PreviewConfig represents configuration for preview generation
type PreviewConfig struct {
	ThumbnailCount int     // Number of thumbnails to generate
	GIFDuration    float64 // Duration of animated GIF in seconds
	GIFStartTime   float64 // Start time for GIF in seconds
	ThumbnailWidth int     // Width of thumbnails (height auto-calculated)
	GIFWidth       int     // Width of GIF (height auto-calculated)
}

// PreviewOutput represents generated preview files
type PreviewOutput struct {
	Thumbnails []string `json:"thumbnails"`
	GIF        string   `json:"gif,omitempty"`
}

func NewPreviewGenerator(cfg *config.Config) *PreviewGenerator {
	return &PreviewGenerator{
		config: cfg,
	}
}

// GeneratePreviews generates thumbnails and animated GIF for a video
func (g *PreviewGenerator) GeneratePreviews(inputPath string, outputDir string, config PreviewConfig) (*PreviewOutput, error) {
	output := &PreviewOutput{
		Thumbnails: []string{},
	}

	// Create preview directory
	previewDir := filepath.Join(outputDir, "previews")
	if err := os.MkdirAll(previewDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create preview directory: %w", err)
	}

	// Generate thumbnails
	if config.ThumbnailCount > 0 {
		thumbnails, err := g.generateThumbnails(inputPath, previewDir, config)
		if err != nil {
			return nil, fmt.Errorf("failed to generate thumbnails: %w", err)
		}
		output.Thumbnails = thumbnails
	}

	// Generate animated GIF
	if config.GIFDuration > 0 {
		gifPath, err := g.generateGIF(inputPath, previewDir, config)
		if err != nil {
			return nil, fmt.Errorf("failed to generate GIF: %w", err)
		}
		output.GIF = gifPath
	}

	return output, nil
}

// generateThumbnails generates multiple thumbnails at intervals
func (g *PreviewGenerator) generateThumbnails(inputPath string, outputDir string, config PreviewConfig) ([]string, error) {
	thumbnails := []string{}
	width := config.ThumbnailWidth
	if width == 0 {
		width = 320 // Default width
	}

	// Get video duration first
	probe := NewProbe(g.config)
	videoInfo, err := probe.GetVideoInfo(inputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to probe video: %w", err)
	}

	// Calculate interval between thumbnails
	interval := videoInfo.Duration / float64(config.ThumbnailCount+1)

	for i := 1; i <= config.ThumbnailCount; i++ {
		timestamp := interval * float64(i)
		thumbnailPath := filepath.Join(outputDir, fmt.Sprintf("thumb_%d.jpg", i))

		// FFmpeg command to extract frame
		cmd := exec.Command("ffmpeg",
			"-ss", fmt.Sprintf("%.2f", timestamp),
			"-i", inputPath,
			"-vframes", "1",
			"-vf", fmt.Sprintf("scale=%d:-1", width),
			"-q:v", "2",
			"-y",
			thumbnailPath,
		)

		if err := cmd.Run(); err != nil {
			return nil, fmt.Errorf("failed to generate thumbnail %d: %w", i, err)
		}

		thumbnails = append(thumbnails, thumbnailPath)
	}

	return thumbnails, nil
}

// generateGIF generates an animated GIF preview
func (g *PreviewGenerator) generateGIF(inputPath string, outputDir string, config PreviewConfig) (string, error) {
	gifPath := filepath.Join(outputDir, "preview.gif")
	width := config.GIFWidth
	if width == 0 {
		width = 480 // Default width
	}

	startTime := config.GIFStartTime
	duration := config.GIFDuration
	if duration == 0 {
		duration = 3.0 // Default 3 seconds
	}

	// FFmpeg command to generate GIF with palette for better quality
	paletteFile := filepath.Join(outputDir, "palette.png")

	// Generate palette
	paletteCmd := exec.Command("ffmpeg",
		"-ss", fmt.Sprintf("%.2f", startTime),
		"-t", fmt.Sprintf("%.2f", duration),
		"-i", inputPath,
		"-vf", fmt.Sprintf("fps=10,scale=%d:-1:flags=lanczos,palettegen", width),
		"-y",
		paletteFile,
	)

	if err := paletteCmd.Run(); err != nil {
		return "", fmt.Errorf("failed to generate palette: %w", err)
	}
	defer os.Remove(paletteFile)

	// Generate GIF using palette
	gifCmd := exec.Command("ffmpeg",
		"-ss", fmt.Sprintf("%.2f", startTime),
		"-t", fmt.Sprintf("%.2f", duration),
		"-i", inputPath,
		"-i", paletteFile,
		"-filter_complex", fmt.Sprintf("fps=10,scale=%d:-1:flags=lanczos[x];[x][1:v]paletteuse", width),
		"-y",
		gifPath,
	)

	if err := gifCmd.Run(); err != nil {
		return "", fmt.Errorf("failed to generate GIF: %w", err)
	}

	return gifPath, nil
}

// GenerateSingleThumbnail generates a single thumbnail at a specific timestamp
func (g *PreviewGenerator) GenerateSingleThumbnail(inputPath string, outputPath string, timestamp float64, width int) error {
	if width == 0 {
		width = 320
	}

	cmd := exec.Command("ffmpeg",
		"-ss", fmt.Sprintf("%.2f", timestamp),
		"-i", inputPath,
		"-vframes", "1",
		"-vf", fmt.Sprintf("scale=%d:-1", width),
		"-q:v", "2",
		"-y",
		outputPath,
	)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to generate thumbnail: %w", err)
	}

	return nil
}

// GenerateSpriteSheet generates a sprite sheet of thumbnails
func (g *PreviewGenerator) GenerateSpriteSheet(inputPath string, outputPath string, config PreviewConfig) error {
	width := config.ThumbnailWidth
	if width == 0 {
		width = 160
	}

	// Get video duration
	probe := NewProbe(g.config)
	videoInfo, err := probe.GetVideoInfo(inputPath)
	if err != nil {
		return fmt.Errorf("failed to probe video: %w", err)
	}

	count := config.ThumbnailCount
	if count == 0 {
		count = 10
	}

	// Calculate interval
	interval := videoInfo.Duration / float64(count+1)

	// Generate sprite sheet with 5 columns
	cols := 5
	rows := (count + cols - 1) / cols

	// FFmpeg filter for sprite sheet
	filter := fmt.Sprintf("fps=1/%f,scale=%d:-1,tile=%dx%d", interval, width, cols, rows)

	cmd := exec.Command("ffmpeg",
		"-i", inputPath,
		"-vf", filter,
		"-frames:v", "1",
		"-q:v", "2",
		"-y",
		outputPath,
	)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to generate sprite sheet: %w", err)
	}

	return nil
}

// GeneratePreviewClip generates a short preview video clip
func (g *PreviewGenerator) GeneratePreviewClip(inputPath string, outputPath string, startTime, duration float64) error {
	if duration == 0 {
		duration = 3.0 // Default 3 seconds
	}

	cmd := exec.Command("ffmpeg",
		"-ss", fmt.Sprintf("%.2f", startTime),
		"-t", fmt.Sprintf("%.2f", duration),
		"-i", inputPath,
		"-c:v", "libx264",
		"-preset", "fast",
		"-crf", "23",
		"-c:a", "aac",
		"-b:a", "128k",
		"-movflags", "+faststart",
		"-y",
		outputPath,
	)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to generate preview clip: %w", err)
	}

	return nil
}
