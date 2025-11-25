package types

import "time"

type JobStatus string

const (
	StatusQueued         JobStatus = "QUEUED"
	StatusRunning        JobStatus = "RUNNING"
	StatusCompleted      JobStatus = "COMPLETED"
	StatusPartialSuccess JobStatus = "PARTIAL_SUCCESS"
	StatusFailed         JobStatus = "FAILED"
	StatusInterrupted    JobStatus = "INTERRUPTED"
)

type Job struct {
	ID            string         `json:"jobId"`
	Status        JobStatus      `json:"status"`
	InputPath     string         `json:"inputPath,omitempty"`
	InputHash     string         `json:"inputHash,omitempty"`
	ChunkDuration int            `json:"chunkDuration,omitempty"`
	Input         *VideoMetadata `json:"input,omitempty"`
	Output        *OutputInfo    `json:"output,omitempty"`
	Progress      *Progress      `json:"progress,omitempty"`
	Meta          Meta           `json:"meta"`
	Retries       int            `json:"retries,omitempty"`
	MaxRetries    int            `json:"maxRetries,omitempty"`
	Error         *JobError      `json:"error,omitempty"`
	Message       string         `json:"message,omitempty"`
}

type VideoMetadata struct {
	FileName           string            `json:"fileName"`
	SourcePath         string            `json:"sourcePath"`
	Resolution         string            `json:"resolution"`
	Duration           float64           `json:"duration"`
	GeneratedQualities []QualityMetadata `json:"generatedQualities,omitempty"`
}

type QualityMetadata struct {
	Quality    string `json:"quality"`
	Resolution string `json:"resolution"`
	Bitrate    string `json:"bitrate"`
}

type OutputInfo struct {
	OutputDir      string          `json:"outputDir"`
	MasterPlaylist string          `json:"masterPlaylist,omitempty"`
	Qualities      []QualityOutput `json:"qualities,omitempty"`
	Thumbnails     []string        `json:"thumbnails,omitempty"`
	SpriteSheet    string          `json:"spriteSheet,omitempty"`
	PreviewClip    string          `json:"previewClip,omitempty"`
	PreviewGIF     string          `json:"previewGif,omitempty"`
	Audio          *AudioInfo      `json:"audio,omitempty"`
}

type AudioInfo struct {
	Normalized bool   `json:"normalized"`
	Standard   string `json:"standard,omitempty"`
}

// VideoTrack represents a video quality track (video-only)
type VideoTrack struct {
	Quality    string `json:"quality"`
	Resolution string `json:"resolution"`
	Bandwidth  int    `json:"bandwidth"`
	Playlist   string `json:"playlist"`
}

// AudioTrack represents an audio track
type AudioTrack struct {
	Language  string `json:"language"`
	Bitrate   string `json:"bitrate"`
	Playlist  string `json:"playlist"`
	IsDefault bool   `json:"isDefault"`
}

type QualityOutput struct {
	Quality    string `json:"quality"`
	Playlist   string `json:"playlist"`
	ChunksDir  string `json:"chunksDir"`
	ChunkCount int    `json:"chunkCount"`
}

type Progress struct {
	Percentage  int    `json:"percentage"`
	CurrentStep string `json:"currentStep"`
}

type Meta struct {
	CreatedAt         time.Time  `json:"createdAt"`
	CompletedAt       *time.Time `json:"completedAt,omitempty"`
	ProcessingTimeSec float64    `json:"processingTimeSec,omitempty"`
}

type JobError struct {
	Code              string   `json:"code"`
	Stage             string   `json:"stage"`
	Message           string   `json:"message"`
	LogsPath          string   `json:"logsPath,omitempty"`
	FailedResolutions []string `json:"failedResolutions,omitempty"`
}

// DetailedVideoInfo contains comprehensive video metadata
type DetailedVideoInfo struct {
	Width           int     `json:"width"`
	Height          int     `json:"height"`
	Duration        float64 `json:"duration"`
	VideoCodec      string  `json:"videoCodec"`
	AudioCodec      string  `json:"audioCodec,omitempty"`
	Bitrate         string  `json:"bitrate"`
	VideoBitrate    string  `json:"videoBitrate"`
	FrameRate       string  `json:"frameRate"`
	HasAudio        bool    `json:"hasAudio"`
	AudioChannels   int     `json:"audioChannels,omitempty"`
	AudioSampleRate string  `json:"audioSampleRate,omitempty"`
	FileSize        string  `json:"fileSize"`
}

// WebhookConfig represents webhook configuration for a job
type WebhookConfig struct {
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
	Secret  string            `json:"secret,omitempty"`
	Retries int               `json:"retries,omitempty"`
}

// ScheduledAt represents scheduled execution time
type ScheduledAt struct {
	Time time.Time `json:"time"`
}

// OutputFormat represents desired output format
type OutputFormat string

const (
	OutputFormatHLS  OutputFormat = "hls"
	OutputFormatMP4  OutputFormat = "mp4"
	OutputFormatWebM OutputFormat = "webm"
)

// PreviewConfig represents preview generation configuration
type PreviewConfig struct {
	Generate       bool    `json:"generate"`
	ThumbnailCount int     `json:"thumbnailCount,omitempty"`
	GIFDuration    float64 `json:"gifDuration,omitempty"`
	GIFStartTime   float64 `json:"gifStartTime,omitempty"`
}

// StorageConfig represents cloud storage configuration
type StorageConfig struct {
	Type   string `json:"type"` // s3, gcs, azure, local
	Bucket string `json:"bucket,omitempty"`
	Region string `json:"region,omitempty"`
	Path   string `json:"path,omitempty"`
}

// CDNConfig represents CDN configuration
type CDNConfig struct {
	Enabled  bool   `json:"enabled"`
	Provider string `json:"provider"` // cloudflare, cloudfront, etc.
	BaseURL  string `json:"baseUrl"`
}

// JobStatusScheduled represents a scheduled job
const JobStatusScheduled JobStatus = "SCHEDULED"
