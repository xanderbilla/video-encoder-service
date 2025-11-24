package types

import "time"

// RetryStrategy defines retry behavior for different error types
type RetryStrategy struct {
	ShouldRetry bool
	MaxAttempts int
	Backoff     []time.Duration
}

// GetRetryStrategy returns the retry strategy for a given error code
func GetRetryStrategy(errorCode string) RetryStrategy {
	switch errorCode {
	// FFmpeg errors - retry with exponential backoff
	case "ERR_FFMPEG_FAILED",
		"ERR_ENCODING_240P_FAILED",
		"ERR_ENCODING_360P_FAILED",
		"ERR_ENCODING_480P_FAILED",
		"ERR_ENCODING_720P_FAILED",
		"ERR_ENCODING_1080P_FAILED",
		"ERR_ENCODING_1440P_FAILED",
		"ERR_ENCODING_4K_FAILED":
		return RetryStrategy{
			ShouldRetry: true,
			MaxAttempts: 3,
			Backoff:     []time.Duration{2 * time.Second, 5 * time.Second, 10 * time.Second},
		}

	// I/O errors - retry with longer backoff
	case "ERR_IO_ERROR":
		return RetryStrategy{
			ShouldRetry: true,
			MaxAttempts: 3,
			Backoff:     []time.Duration{3 * time.Second, 6 * time.Second, 15 * time.Second},
		}

	// Invalid input - no retry
	case "ERR_INVALID_VIDEO",
		"ERR_UNSUPPORTED_FORMAT",
		"ERR_CORRUPTED_FILE",
		"ERR_NO_VIDEO_STREAM",
		"ERR_RESOLUTION_TOO_HIGH",
		"ERR_DURATION_TOO_SHORT",
		"ERR_RESOLUTION_DETECT_FAILED":
		return RetryStrategy{
			ShouldRetry: false,
			MaxAttempts: 0,
			Backoff:     []time.Duration{},
		}

	// Disk full - no retry
	case "ERR_DISK_FULL",
		"ERR_JOB_SIZE_EXCEEDED":
		return RetryStrategy{
			ShouldRetry: false,
			MaxAttempts: 0,
			Backoff:     []time.Duration{},
		}

	// Default - basic retry
	default:
		return RetryStrategy{
			ShouldRetry: true,
			MaxAttempts: 2,
			Backoff:     []time.Duration{2 * time.Second, 5 * time.Second},
		}
	}
}
