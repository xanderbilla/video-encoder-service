package filesystem

import (
	"encoder-service/internal/core/domain"
	"encoder-service/internal/infrastructure/ffmpeg"
	"encoder-service/pkg/constants"
	"encoder-service/pkg/utils"
	"fmt"
	"path/filepath"
	"strings"
)

type FileValidator struct {
	allowedExtensions map[string]bool
}

func NewFileValidator() *FileValidator {
	return &FileValidator{
		allowedExtensions: map[string]bool{
			".mp4":  true,
			".mkv":  true,
			".mov":  true,
			".webm": true,
			".avi":  true,
		},
	}
}

type ValidationError struct {
	Code    string
	Message string
	Details map[string]interface{}
}

func (e *ValidationError) Error() string {
	return e.Message
}

func (v *FileValidator) ValidateFile(filePath string) error {
	// Check file exists
	if !utils.FileExists(filePath) {
		return &ValidationError{
			Code:    domain.CodeFileNotFound,
			Message: "File not found",
			Details: map[string]interface{}{"path": filePath},
		}
	}

	// Check file size
	fileSizeMB, err := utils.GetFileSizeMB(filePath)
	if err != nil {
		return &ValidationError{
			Code:    domain.CodeInvalidVideo,
			Message: "Failed to get file size",
			Details: map[string]interface{}{"error": err.Error()},
		}
	}

	if fileSizeMB > float64(constants.MaxFileSizeMB) {
		return &ValidationError{
			Code:    domain.CodeFileTooLarge,
			Message: fmt.Sprintf("File size exceeds maximum allowed size of %dMB", constants.MaxFileSizeMB),
			Details: map[string]interface{}{
				"fileSizeMB": fileSizeMB,
				"maxSizeMB":  constants.MaxFileSizeMB,
			},
		}
	}

	// Check file extension
	ext := strings.ToLower(filepath.Ext(filePath))
	if !v.allowedExtensions[ext] {
		return &ValidationError{
			Code:    domain.CodeUnsupportedFormat,
			Message: "File format not supported",
			Details: map[string]interface{}{
				"extension": ext,
				"allowed":   []string{".mp4", ".mkv", ".mov", ".webm", ".avi"},
			},
		}
	}

	// Validate MIME type
	if err := v.ValidateMimeType(filePath); err != nil {
		return err
	}

	// Validate video stream
	if err := v.ValidateVideoStream(filePath); err != nil {
		return err
	}

	// Validate resolution
	if err := v.ValidateResolution(filePath); err != nil {
		return err
	}

	// Validate duration
	if err := v.ValidateDuration(filePath); err != nil {
		return err
	}

	// Test decode
	if err := v.TestDecode(filePath); err != nil {
		return &ValidationError{
			Code:    domain.CodeCorruptedFile,
			Message: "Video file appears to be corrupted",
			Details: map[string]interface{}{"error": err.Error()},
		}
	}

	return nil
}

func (v *FileValidator) ValidateVideoStream(filePath string) error {
	if err := ffmpeg.ValidateVideoStream(filePath); err != nil {
		return &ValidationError{
			Code:    domain.CodeNoVideoStream,
			Message: "File does not contain a valid video stream",
			Details: map[string]interface{}{"error": err.Error()},
		}
	}
	return nil
}

func (v *FileValidator) TestDecode(filePath string) error {
	return ffmpeg.TestDecode(filePath)
}

func (v *FileValidator) ValidateResolution(filePath string) error {
	videoInfo, err := ffmpeg.GetVideoInfo(filePath)
	if err != nil {
		return &ValidationError{
			Code:    domain.CodeInvalidVideo,
			Message: "Failed to get video resolution",
			Details: map[string]interface{}{"error": err.Error()},
		}
	}

	if videoInfo.Width > constants.MaxWidth || videoInfo.Height > constants.MaxHeight {
		return &ValidationError{
			Code:    "ERR_RESOLUTION_TOO_HIGH",
			Message: fmt.Sprintf("Video resolution exceeds maximum allowed (%dx%d)", constants.MaxWidth, constants.MaxHeight),
			Details: map[string]interface{}{
				"width":     videoInfo.Width,
				"height":    videoInfo.Height,
				"maxWidth":  constants.MaxWidth,
				"maxHeight": constants.MaxHeight,
			},
		}
	}

	return nil
}

func (v *FileValidator) ValidateDuration(filePath string) error {
	videoInfo, err := ffmpeg.GetVideoInfo(filePath)
	if err != nil {
		return &ValidationError{
			Code:    domain.CodeInvalidVideo,
			Message: "Failed to get video duration",
			Details: map[string]interface{}{"error": err.Error()},
		}
	}

	if videoInfo.Duration < constants.MinDuration {
		return &ValidationError{
			Code:    "ERR_DURATION_TOO_SHORT",
			Message: fmt.Sprintf("Video duration must be at least %.1f seconds", constants.MinDuration),
			Details: map[string]interface{}{
				"duration":    videoInfo.Duration,
				"minDuration": constants.MinDuration,
			},
		}
	}

	return nil
}

func (v *FileValidator) ValidateMimeType(filePath string) error {
	// Use ffprobe to get actual format
	format, err := ffmpeg.GetFileFormat(filePath)
	if err != nil {
		return &ValidationError{
			Code:    domain.CodeInvalidVideo,
			Message: "Failed to detect file format",
			Details: map[string]interface{}{"error": err.Error()},
		}
	}

	// Check if format is in allowed list
	allowedFormats := map[string]bool{
		"mov,mp4,m4a,3gp,3g2,mj2": true, // MP4/MOV
		"matroska,webm":           true, // MKV/WebM
		"avi":                     true, // AVI
	}

	if !allowedFormats[format] {
		return &ValidationError{
			Code:    domain.CodeUnsupportedFormat,
			Message: "File format not supported",
			Details: map[string]interface{}{
				"detectedFormat": format,
				"allowed":        []string{"mp4", "mov", "mkv", "webm", "avi"},
			},
		}
	}

	return nil
}
