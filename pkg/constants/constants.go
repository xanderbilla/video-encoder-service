package constants

// File size limits (in MB)
const (
	MaxFileSizeMB = 2048 // 2GB
	MaxJobSizeMB  = 3000 // 3GB per job
)

// Resolution limits
const (
	MaxWidth  = 3840 // 4K
	MaxHeight = 2160 // 4K
)

// Duration limits (in seconds)
const (
	MinDuration        = 0.1 // 100ms minimum
	DefaultChunkDuration = 4   // 4 seconds
)

// Disk management
const (
	DiskThresholdPercent = 80 // Stop accepting new jobs
	HardLimitPercent     = 90 // Auto cleanup
)

// Worker configuration
const (
	MaxWorkers        = 2
	MaxRetries        = 3
	HeartbeatInterval = 10 // seconds
	OrphanTimeout     = 60 // seconds
)

// Timeouts (in seconds)
const (
	GracefulShutdownTimeout = 30
	CleanupInterval         = 600 // 10 minutes
)

// FFmpeg settings
const (
	FFmpegAudioCodec     = "aac"
	FFmpegAudioSampleRate = "48000"
	FFmpegAudioBitrate   = "128k"
	FFmpegVideoCodec     = "libx264"
	FFmpegHLSPlaylistType = "vod"
)

// Directory names
const (
	OutputsDir = "outputs"
	ChunksDir  = "chunks"
	CacheDir   = "cache"
	TmpDir     = "tmp"
	VideosDir  = "videos"
	LogsDir    = "logs"
	HLSDir     = "hls"
)

// File patterns
const (
	ChunkPattern    = "chunk_%03d.ts"
	PlaylistName    = "main.m3u8"
	MasterPlaylist  = "master.m3u8"
)
