package services

import (
	"encoder-service/pkg/config"
	"encoder-service/pkg/utils"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type DiskService struct {
	config *config.Config
}

func NewDiskService(cfg *config.Config) *DiskService {
	return &DiskService{
		config: cfg,
	}
}

func (s *DiskService) CanAcceptNewJob() (bool, string) {
	usage, err := utils.GetDiskUsagePercent(".")
	if err != nil {
		return false, fmt.Sprintf("Failed to check disk usage: %v", err)
	}

	if usage >= float64(s.config.Disk.ThresholdPercent) {
		return false, fmt.Sprintf("Disk usage at %.1f%%, exceeds threshold of %d%%", usage, s.config.Disk.ThresholdPercent)
	}

	return true, ""
}

func (s *DiskService) AutoCleanup() error {
	usage, err := utils.GetDiskUsagePercent(".")
	if err != nil {
		return err
	}

	if usage < float64(s.config.Disk.HardLimitPercent) {
		return nil
	}

	log.Printf("Disk usage at %.1f%%, starting auto cleanup...", usage)

	jobs, err := s.getJobsByAge()
	if err != nil {
		return err
	}

	cleaned := 0
	for _, jobInfo := range jobs {
		currentUsage, _ := utils.GetDiskUsagePercent(".")
		if currentUsage < float64(s.config.Disk.ThresholdPercent) {
			break
		}

		log.Printf("Cleaning up old job: %s (age: %v)", jobInfo.JobID, time.Since(jobInfo.ModTime))

		if err := s.DeleteJobFiles(jobInfo.JobID); err != nil {
			log.Printf("Failed to delete job %s: %v", jobInfo.JobID, err)
			continue
		}

		cleaned++
	}

	log.Printf("Auto cleanup completed, removed %d jobs", cleaned)
	return nil
}

func (s *DiskService) DeleteJobFiles(jobID string) error {
	outputDir := filepath.Join(s.config.Storage.OutputsDir, jobID)
	chunkDir := filepath.Join(s.config.Storage.ChunksDir, jobID)

	if err := utils.RemoveDir(outputDir); err != nil {
		return fmt.Errorf("failed to delete output dir: %w", err)
	}

	if err := utils.RemoveDir(chunkDir); err != nil {
		return fmt.Errorf("failed to delete chunk dir: %w", err)
	}

	return nil
}

type JobInfo struct {
	JobID   string
	ModTime time.Time
	Size    int64
}

func (s *DiskService) getJobsByAge() ([]JobInfo, error) {
	var jobs []JobInfo

	entries, err := os.ReadDir(s.config.Storage.OutputsDir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		size, _ := utils.GetDirSize(filepath.Join(s.config.Storage.OutputsDir, entry.Name()))

		jobs = append(jobs, JobInfo{
			JobID:   entry.Name(),
			ModTime: info.ModTime(),
			Size:    size,
		})
	}

	sort.Slice(jobs, func(i, j int) bool {
		return jobs[i].ModTime.Before(jobs[j].ModTime)
	})

	return jobs, nil
}

func (s *DiskService) StartCleanupWorker() {
	ticker := time.NewTicker(time.Duration(s.config.Disk.CleanupInterval) * time.Second)
	go func() {
		for range ticker.C {
			if err := s.AutoCleanup(); err != nil {
				log.Printf("Auto cleanup error: %v", err)
			}
		}
	}()
}

func (s *DiskService) GetJobSize(jobID string) (int64, error) {
	var totalSize int64

	// Check outputs directory
	outputDir := filepath.Join(s.config.Storage.OutputsDir, jobID)
	if utils.FileExists(outputDir) {
		size, err := utils.GetDirSize(outputDir)
		if err != nil {
			return 0, fmt.Errorf("failed to get output dir size: %w", err)
		}
		totalSize += size
	}

	// Check chunks directory
	chunkDir := filepath.Join(s.config.Storage.ChunksDir, jobID)
	if utils.FileExists(chunkDir) {
		size, err := utils.GetDirSize(chunkDir)
		if err != nil {
			return 0, fmt.Errorf("failed to get chunk dir size: %w", err)
		}
		totalSize += size
	}

	return totalSize, nil
}

func (s *DiskService) CleanupIntermediateFiles(jobID string) error {
	// Remove chunks directory (temporary files)
	chunkDir := filepath.Join(s.config.Storage.ChunksDir, jobID)
	if utils.FileExists(chunkDir) {
		if err := utils.RemoveDir(chunkDir); err != nil {
			return fmt.Errorf("failed to remove chunks: %w", err)
		}
		log.Printf("Cleaned up chunks for job %s", jobID)
	}

	// Remove tmp files if any
	tmpDir := filepath.Join(s.config.Storage.TmpDir, jobID)
	if utils.FileExists(tmpDir) {
		if err := utils.RemoveDir(tmpDir); err != nil {
			return fmt.Errorf("failed to remove tmp files: %w", err)
		}
		log.Printf("Cleaned up tmp files for job %s", jobID)
	}

	return nil
}

func (s *DiskService) GetOutputsDir() string {
	return s.config.Storage.OutputsDir
}
