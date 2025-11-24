package config

import (
	"encoder-service/pkg/constants"
	"os"
	"strconv"
)

type Config struct {
	Server   ServerConfig
	Worker   WorkerConfig
	Storage  StorageConfig
	Encoding EncodingConfig
	Disk     DiskConfig
}

type ServerConfig struct {
	Port            string
	ShutdownTimeout int
}

type WorkerConfig struct {
	MaxWorkers        int
	MaxRetries        int
	HeartbeatInterval int
	OrphanTimeout     int
}

type StorageConfig struct {
	OutputsDir string
	ChunksDir  string
	CacheDir   string
	TmpDir     string
	VideosDir  string
	LogsDir    string
}

type EncodingConfig struct {
	MaxFileSizeMB        int
	MaxJobSizeMB         int
	MaxWidth             int
	MaxHeight            int
	DefaultChunkDuration int
}

type DiskConfig struct {
	ThresholdPercent int
	HardLimitPercent int
	CleanupInterval  int
}

func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Port:            getEnv("PORT", "8080"),
			ShutdownTimeout: getEnvInt("SHUTDOWN_TIMEOUT", constants.GracefulShutdownTimeout),
		},
		Worker: WorkerConfig{
			MaxWorkers:        getEnvInt("MAX_WORKERS", constants.MaxWorkers),
			MaxRetries:        getEnvInt("MAX_RETRIES", constants.MaxRetries),
			HeartbeatInterval: getEnvInt("HEARTBEAT_INTERVAL", constants.HeartbeatInterval),
			OrphanTimeout:     getEnvInt("ORPHAN_TIMEOUT", constants.OrphanTimeout),
		},
		Storage: StorageConfig{
			OutputsDir: getEnv("OUTPUTS_DIR", constants.OutputsDir),
			ChunksDir:  getEnv("CHUNKS_DIR", constants.ChunksDir),
			CacheDir:   getEnv("CACHE_DIR", constants.CacheDir),
			TmpDir:     getEnv("TMP_DIR", constants.TmpDir),
			VideosDir:  getEnv("VIDEOS_DIR", constants.VideosDir),
			LogsDir:    getEnv("LOGS_DIR", constants.LogsDir),
		},
		Encoding: EncodingConfig{
			MaxFileSizeMB:        getEnvInt("MAX_FILE_SIZE_MB", constants.MaxFileSizeMB),
			MaxJobSizeMB:         getEnvInt("MAX_JOB_SIZE_MB", constants.MaxJobSizeMB),
			MaxWidth:             getEnvInt("MAX_WIDTH", constants.MaxWidth),
			MaxHeight:            getEnvInt("MAX_HEIGHT", constants.MaxHeight),
			DefaultChunkDuration: getEnvInt("DEFAULT_CHUNK_DURATION", constants.DefaultChunkDuration),
		},
		Disk: DiskConfig{
			ThresholdPercent: getEnvInt("DISK_THRESHOLD_PERCENT", constants.DiskThresholdPercent),
			HardLimitPercent: getEnvInt("DISK_HARD_LIMIT_PERCENT", constants.HardLimitPercent),
			CleanupInterval:  getEnvInt("CLEANUP_INTERVAL", constants.CleanupInterval),
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
