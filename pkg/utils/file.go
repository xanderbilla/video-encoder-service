package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// ComputeFileHash calculates SHA256 hash of a file
func ComputeFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("failed to compute hash: %w", err)
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// GetFileSize returns file size in bytes
func GetFileSize(filePath string) (int64, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return 0, fmt.Errorf("failed to get file info: %w", err)
	}
	return info.Size(), nil
}

// GetFileSizeMB returns file size in megabytes
func GetFileSizeMB(filePath string) (float64, error) {
	size, err := GetFileSize(filePath)
	if err != nil {
		return 0, err
	}
	return float64(size) / (1024 * 1024), nil
}

// EnsureDir creates directory if it doesn't exist
func EnsureDir(dirPath string) error {
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	return nil
}

// GetDirSize calculates total size of directory
func GetDirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size, err
}

// FileExists checks if file exists
func FileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return err == nil
}

// RemoveDir removes directory and all its contents
func RemoveDir(dirPath string) error {
	if !FileExists(dirPath) {
		return nil
	}
	return os.RemoveAll(dirPath)
}
