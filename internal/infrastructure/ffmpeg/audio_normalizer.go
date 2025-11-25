package ffmpeg

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// AudioNormalizer handles audio loudness normalization
type AudioNormalizer struct{}

// LoudnessMetrics contains measured audio loudness values
type LoudnessMetrics struct {
	InputI       string `json:"input_i"`
	InputTP      string `json:"input_tp"`
	InputLRA     string `json:"input_lra"`
	InputThresh  string `json:"input_thresh"`
	OutputI      string `json:"output_i"`
	OutputTP     string `json:"output_tp"`
	OutputLRA    string `json:"output_lra"`
	OutputThresh string `json:"output_thresh"`
	Offset       string `json:"offset"`
}

func NewAudioNormalizer() *AudioNormalizer {
	return &AudioNormalizer{}
}

// MeasureLoudness analyzes audio loudness using ITU BS.1770 standard
func (n *AudioNormalizer) MeasureLoudness(inputPath string) (*LoudnessMetrics, error) {
	cmd := exec.Command("ffmpeg",
		"-i", inputPath,
		"-af", "loudnorm=I=-16:TP=-1.5:LRA=11:print_format=json",
		"-f", "null",
		"-",
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to measure loudness: %w", err)
	}

	// Extract JSON from output
	outputStr := string(output)
	jsonStart := strings.Index(outputStr, "{")
	if jsonStart == -1 {
		return nil, fmt.Errorf("no JSON found in loudness output")
	}

	var metrics LoudnessMetrics
	if err := json.Unmarshal([]byte(outputStr[jsonStart:]), &metrics); err != nil {
		return nil, fmt.Errorf("failed to parse loudness metrics: %w", err)
	}

	return &metrics, nil
}

// NormalizeAudio applies loudness normalization to audio
func (n *AudioNormalizer) NormalizeAudio(inputPath, outputPath string, metrics *LoudnessMetrics) error {
	filterComplex := fmt.Sprintf(
		"loudnorm=I=-16:TP=-1.5:LRA=11:measured_I=%s:measured_TP=%s:measured_LRA=%s:measured_thresh=%s:offset=%s:linear=true:print_format=summary",
		metrics.InputI,
		metrics.InputTP,
		metrics.InputLRA,
		metrics.InputThresh,
		metrics.Offset,
	)

	cmd := exec.Command("ffmpeg",
		"-i", inputPath,
		"-af", filterComplex,
		"-c:v", "copy",
		"-c:a", "aac",
		"-b:a", "192k",
		"-y",
		outputPath,
	)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to normalize audio: %w", err)
	}

	return nil
}

// NormalizeAudioInPlace normalizes audio and replaces original
func (n *AudioNormalizer) NormalizeAudioInPlace(inputPath, tempDir string) (string, error) {
	// Measure loudness
	metrics, err := n.MeasureLoudness(inputPath)
	if err != nil {
		return "", err
	}

	// Create normalized output
	normalizedPath := filepath.Join(tempDir, "normalized_"+filepath.Base(inputPath))
	if err := n.NormalizeAudio(inputPath, normalizedPath, metrics); err != nil {
		return "", err
	}

	return normalizedPath, nil
}
