package domain

import "errors"

// Domain errors
var (
	ErrJobNotFound      = errors.New("job not found")
	ErrJobAlreadyExists = errors.New("job already exists")
	ErrInvalidJobStatus = errors.New("invalid job status")
	ErrFileNotFound     = errors.New("file not found")
	ErrInvalidFile      = errors.New("invalid file")
	ErrFileTooLarge     = errors.New("file too large")
	ErrUnsupportedFormat = errors.New("unsupported format")
	ErrCorruptedFile    = errors.New("corrupted file")
	ErrNoVideoStream    = errors.New("no video stream found")
	ErrEncodingFailed   = errors.New("encoding failed")
	ErrDiskFull         = errors.New("disk full")
	ErrInvalidInput     = errors.New("invalid input")
)

// Error codes for API responses
const (
	// API Validation Errors
	CodeJobNotFound        = "ERR_JOB_NOT_FOUND"
	CodeInvalidRequest     = "ERR_INVALID_REQUEST"
	CodeMissingField       = "ERR_MISSING_FIELD"
	CodeFileNotFound       = "ERR_FILE_NOT_FOUND"
	CodeFileTooLarge       = "ERR_FILE_TOO_LARGE"
	CodeUnsupportedFormat  = "ERR_UNSUPPORTED_FORMAT"
	CodeCorruptedFile      = "ERR_CORRUPTED_FILE"
	CodeNoVideoStream      = "ERR_NO_VIDEO_STREAM"
	CodeInvalidVideo       = "ERR_INVALID_VIDEO"
	CodeResolutionTooHigh  = "ERR_RESOLUTION_TOO_HIGH"
	CodeDurationTooShort   = "ERR_DURATION_TOO_SHORT"

	// Worker/FFmpeg Errors
	CodeEncodingFailed           = "ERR_ENCODING_FAILED"
	CodeFFmpegFailed             = "ERR_FFMPEG_FAILED"
	CodeResolutionDetectFailed   = "ERR_RESOLUTION_DETECT_FAILED"
	CodeEncoding240pFailed       = "ERR_ENCODING_240P_FAILED"
	CodeEncoding360pFailed       = "ERR_ENCODING_360P_FAILED"
	CodeEncoding480pFailed       = "ERR_ENCODING_480P_FAILED"
	CodeEncoding720pFailed       = "ERR_ENCODING_720P_FAILED"
	CodeEncoding1080pFailed      = "ERR_ENCODING_1080P_FAILED"
	CodeEncoding1440pFailed      = "ERR_ENCODING_1440P_FAILED"
	CodeEncoding4KFailed         = "ERR_ENCODING_4K_FAILED"

	// System Errors
	CodeDiskFull           = "ERR_DISK_FULL"
	CodeIOError            = "ERR_IO_ERROR"
	CodePermissionDenied   = "ERR_PERMISSION_DENIED"
	CodeDeleteFailed       = "ERR_DELETE_FAILED"
	CodeJobSizeExceeded    = "ERR_JOB_SIZE_EXCEEDED"
	CodeInternalError      = "ERR_INTERNAL_ERROR"
)
