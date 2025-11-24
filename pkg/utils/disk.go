package utils

import (
	"fmt"
	"syscall"
)

// GetDiskUsagePercent returns disk usage percentage
func GetDiskUsagePercent(path string) (float64, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, fmt.Errorf("failed to get disk stats: %w", err)
	}

	total := stat.Blocks * uint64(stat.Bsize)
	free := stat.Bfree * uint64(stat.Bsize)
	used := total - free

	usagePercent := float64(used) / float64(total) * 100
	return usagePercent, nil
}

// GetDiskSpace returns total, used, and free disk space in bytes
func GetDiskSpace(path string) (total, used, free uint64, err error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, 0, 0, fmt.Errorf("failed to get disk stats: %w", err)
	}

	total = stat.Blocks * uint64(stat.Bsize)
	free = stat.Bfree * uint64(stat.Bsize)
	used = total - free

	return total, used, free, nil
}
