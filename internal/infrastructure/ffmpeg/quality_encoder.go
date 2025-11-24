package ffmpeg

import (
	"context"
	"encoder-service/pkg/constants"
	"encoder-service/pkg/types"
	"encoder-service/pkg/utils"
	"fmt"
	"io"
	"log"
	"os/exec"
	"path/filepath"
)

// QualityEncodingResult represents the result of encoding a single quality
type QualityEncodingResult struct {
	Quality    string
	Success    bool
	Error      error
	ErrorCode  string
	Playlist   string
	ChunksDir  string
	ChunkCount int
}

// EncodeQuality encodes a single quality variant with watchdog
func EncodeQuality(jobID, inputPath string, variant types.QualityVariant, chunkDuration int, outputsDir string, hasAudio bool) QualityEncodingResult {
	result := QualityEncodingResult{
		Quality: variant.Name,
		Success: false,
	}

	// Create quality-specific directory
	hlsDir := filepath.Join(outputsDir, jobID, constants.HLSDir)
	variantDir := filepath.Join(hlsDir, variant.Name)
	
	if err := utils.EnsureDir(variantDir); err != nil {
		result.Error = fmt.Errorf("failed to create variant dir: %w", err)
		result.ErrorCode = getQualityErrorCode(variant.Name)
		return result
	}

	// Get video duration for watchdog
	videoInfo, err := GetVideoInfo(inputPath)
	if err != nil {
		result.Error = fmt.Errorf("failed to get video info: %w", err)
		result.ErrorCode = getQualityErrorCode(variant.Name)
		return result
	}

	// Create watchdog
	watchdog := utils.NewWatchdog(videoInfo.Duration)
	log.Printf("Encoding %s with watchdog timeout: %v", variant.Name, watchdog.GetTimeout())

	// Execute with watchdog
	err = watchdog.Watch(context.Background(), func() error {
		return executeFFmpegEncoding(jobID, inputPath, variantDir, variant, chunkDuration, hasAudio, outputsDir)
	})

	if err != nil {
		result.Error = err
		result.ErrorCode = getQualityErrorCode(variant.Name)
		return result
	}

	// Count chunks
	chunks, err := filepath.Glob(filepath.Join(variantDir, "chunk_*.ts"))
	if err != nil {
		result.Error = fmt.Errorf("failed to list chunks: %w", err)
		result.ErrorCode = getQualityErrorCode(variant.Name)
		return result
	}

	if len(chunks) == 0 {
		result.Error = fmt.Errorf("no chunks generated")
		result.ErrorCode = getQualityErrorCode(variant.Name)
		return result
	}

	result.Success = true
	result.Playlist = filepath.Join(variantDir, "main.m3u8")
	result.ChunksDir = variantDir
	result.ChunkCount = len(chunks)

	log.Printf("Successfully encoded %s for job %s (%d chunks)", variant.Name, jobID, len(chunks))
	return result
}

func executeFFmpegEncoding(jobID, inputPath, variantDir string, variant types.QualityVariant, chunkDuration int, hasAudio bool, outputsDir string) error {
	// Build FFmpeg command for single quality
	args := buildSingleQualityArgs(inputPath, variantDir, variant, chunkDuration, hasAudio)

	// Execute FFmpeg
	cmd := exec.Command("ffmpeg", args...)
	
	// Capture stderr for logging
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start ffmpeg: %w", err)
	}

	// Save FFmpeg logs
	logPath := filepath.Join(outputsDir, jobID, "logs")
	utils.EnsureDir(logPath)
	saveFFmpegLogs(stderr, filepath.Join(logPath, fmt.Sprintf("%s.log", variant.Name)))

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("ffmpeg encoding failed: %w", err)
	}

	return nil
}

func buildSingleQualityArgs(inputPath, outputDir string, variant types.QualityVariant, chunkDuration int, hasAudio bool) []string {
	args := []string{
		"-i", inputPath,
		"-vf", fmt.Sprintf("scale=%d:%d", variant.Width, variant.Height),
		"-c:v", constants.FFmpegVideoCodec,
		"-b:v", variant.Bitrate,
		"-maxrate", variant.MaxRate,
		"-bufsize", variant.BufSize,
	}

	if hasAudio {
		args = append(args,
			"-c:a", constants.FFmpegAudioCodec,
			"-ar", constants.FFmpegAudioSampleRate,
			"-b:a", constants.FFmpegAudioBitrate,
		)
	}

	args = append(args,
		"-f", "hls",
		"-hls_time", fmt.Sprintf("%d", chunkDuration),
		"-hls_playlist_type", constants.FFmpegHLSPlaylistType,
		"-hls_segment_filename", filepath.Join(outputDir, constants.ChunkPattern),
		"-y",
		filepath.Join(outputDir, "main.m3u8"),
	)

	return args
}

func getQualityErrorCode(quality string) string {
	switch quality {
	case "240p":
		return "ERR_ENCODING_240P_FAILED"
	case "360p":
		return "ERR_ENCODING_360P_FAILED"
	case "480p":
		return "ERR_ENCODING_480P_FAILED"
	case "720p":
		return "ERR_ENCODING_720P_FAILED"
	case "1080p":
		return "ERR_ENCODING_1080P_FAILED"
	case "1440p":
		return "ERR_ENCODING_1440P_FAILED"
	case "4K":
		return "ERR_ENCODING_4K_FAILED"
	default:
		return "ERR_ENCODING_FAILED"
	}
}

func saveFFmpegLogs(stderr io.ReadCloser, logPath string) {
	// Save FFmpeg stderr to log file
	logDir := filepath.Dir(logPath)
	utils.EnsureDir(logDir)
	
	// Read all stderr (simplified - in production you'd want streaming)
	data, err := io.ReadAll(stderr)
	if err != nil {
		log.Printf("Warning: Failed to read FFmpeg stderr: %v", err)
		return
	}
	
	// Write to log file
	if err := utils.WriteFile(logPath, data); err != nil {
		log.Printf("Warning: Failed to write FFmpeg log: %v", err)
	}
}
