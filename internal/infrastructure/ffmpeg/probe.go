package ffmpeg

import (
	"encoder-service/pkg/config"
	"encoder-service/pkg/types"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// Probe handles video analysis and metadata extraction
type Probe struct {
	config *config.Config
}

func NewProbe(cfg *config.Config) *Probe {
	return &Probe{
		config: cfg,
	}
}

// GetVideoInfo retrieves basic video metadata using ffprobe
func (p *Probe) GetVideoInfo(inputPath string) (*types.VideoInfo, error) {
	cmd := exec.Command("ffprobe",
		"-v", "quiet",
		"-select_streams", "v:0",
		"-show_entries", "stream=width,height:format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		inputPath,
	)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get video info: %w", err)
	}

	var info types.VideoInfo
	_, err = fmt.Sscanf(string(output), "%d\n%d\n%f", &info.Width, &info.Height, &info.Duration)
	if err != nil {
		return nil, fmt.Errorf("failed to parse video info: %w", err)
	}

	info.HasAudio = p.HasAudioStream(inputPath)
	return &info, nil
}

// GetDetailedVideoInfo retrieves comprehensive video metadata
func (p *Probe) GetDetailedVideoInfo(inputPath string) (*types.DetailedVideoInfo, error) {
	cmd := exec.Command("ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		inputPath,
	)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get detailed video info: %w", err)
	}

	var probeData struct {
		Streams []struct {
			CodecType    string  `json:"codec_type"`
			CodecName    string  `json:"codec_name"`
			Width        int     `json:"width"`
			Height       int     `json:"height"`
			BitRate      string  `json:"bit_rate"`
			FrameRate    string  `json:"r_frame_rate"`
			Channels     int     `json:"channels"`
			SampleRate   string  `json:"sample_rate"`
			Duration     string  `json:"duration"`
		} `json:"streams"`
		Format struct {
			Duration string `json:"duration"`
			BitRate  string `json:"bit_rate"`
			Size     string `json:"size"`
		} `json:"format"`
	}

	if err := json.Unmarshal(output, &probeData); err != nil {
		return nil, fmt.Errorf("failed to parse probe data: %w", err)
	}

	info := &types.DetailedVideoInfo{}

	// Parse video stream
	for _, stream := range probeData.Streams {
		if stream.CodecType == "video" {
			info.Width = stream.Width
			info.Height = stream.Height
			info.VideoCodec = stream.CodecName
			info.VideoBitrate = stream.BitRate
			info.FrameRate = stream.FrameRate
		} else if stream.CodecType == "audio" {
			info.HasAudio = true
			info.AudioCodec = stream.CodecName
			info.AudioChannels = stream.Channels
			info.AudioSampleRate = stream.SampleRate
		}
	}

	// Parse format
	fmt.Sscanf(probeData.Format.Duration, "%f", &info.Duration)
	info.Bitrate = probeData.Format.BitRate
	info.FileSize = probeData.Format.Size

	return info, nil
}

// HasAudioStream checks if video has audio stream
func (p *Probe) HasAudioStream(inputPath string) bool {
	cmd := exec.Command("ffprobe",
		"-v", "quiet",
		"-select_streams", "a:0",
		"-show_entries", "stream=codec_type",
		"-of", "default=noprint_wrappers=1:nokey=1",
		inputPath,
	)

	output, err := cmd.Output()
	if err != nil {
		return false
	}

	return len(output) > 0 && strings.HasPrefix(string(output), "audio")
}

// Legacy functions for backward compatibility
func GetVideoInfo(inputPath string) (*types.VideoInfo, error) {
	probe := &Probe{}
	return probe.GetVideoInfo(inputPath)
}

func HasAudioStream(inputPath string) bool {
	probe := &Probe{}
	return probe.HasAudioStream(inputPath)
}

// ValidateVideoStream validates video stream using ffprobe
func ValidateVideoStream(filePath string) error {
	cmd := exec.Command("ffprobe",
		"-v", "error",
		"-show_streams",
		"-show_format",
		"-of", "json",
		filePath,
	)

	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to analyze video file: %w", err)
	}

	if !strings.Contains(string(output), `"codec_type": "video"`) {
		return fmt.Errorf("file does not contain a valid video stream")
	}

	return nil
}

// TestDecode tests if video can be decoded
func TestDecode(filePath string) error {
	cmd := exec.Command("ffmpeg",
		"-v", "error",
		"-i", filePath,
		"-t", "1",
		"-f", "null",
		"-",
	)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("decode test failed: %w", err)
	}

	return nil
}

// GetFileFormat returns the actual file format detected by ffprobe
func GetFileFormat(filePath string) (string, error) {
	cmd := exec.Command("ffprobe",
		"-v", "quiet",
		"-show_entries", "format=format_name",
		"-of", "default=noprint_wrappers=1:nokey=1",
		filePath,
	)

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get file format: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}
