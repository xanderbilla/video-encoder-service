package services

import (
	"encoding/json"
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

	// Check if we need cleanup
	thresholdPercent := float64(s.config.Disk.ThresholdPercent)
	hardLimitPercent := float64(s.config.Disk.HardLimitPercent)

	if usage < thresholdPercent {
		return nil
	}

	// Determine cleanup strategy based on disk usage
	isEmergency := usage >= hardLimitPercent
	targetUsage := thresholdPercent - 5 // Clean until 5% below threshold

	if isEmergency {
		log.Printf("⚠️  EMERGENCY: Disk usage at %.1f%% (hard limit: %.0f%%), starting aggressive cleanup...", 
			usage, hardLimitPercent)
		targetUsage = thresholdPercent - 10 // More aggressive for emergency
	} else {
		log.Printf("Disk usage at %.1f%%, starting auto cleanup...", usage)
	}

	// Get jobs sorted by priority for cleanup
	jobs, err := s.getJobsForCleanup(isEmergency)
	if err != nil {
		return err
	}

	cleaned := 0
	totalFreed := int64(0)

	for _, jobInfo := range jobs {
		currentUsage, _ := utils.GetDiskUsagePercent(".")
		if currentUsage < targetUsage {
			break
		}

		log.Printf("Cleaning up job: %s (age: %v, size: %.2f MB, status: %s)", 
			jobInfo.JobID, time.Since(jobInfo.ModTime), float64(jobInfo.Size)/(1024*1024), jobInfo.Status)

		if err := s.DeleteJobFiles(jobInfo.JobID); err != nil {
			log.Printf("Failed to delete job %s: %v", jobInfo.JobID, err)
			continue
		}

		cleaned++
		totalFreed += jobInfo.Size
	}

	freedMB := float64(totalFreed) / (1024 * 1024)
	finalUsage, _ := utils.GetDiskUsagePercent(".")

	if isEmergency {
		log.Printf("🚨 Emergency cleanup completed: removed %d jobs, freed %.2f MB, disk usage: %.1f%% → %.1f%%", 
			cleaned, freedMB, usage, finalUsage)
	} else {
		log.Printf("Auto cleanup completed: removed %d jobs, freed %.2f MB, disk usage: %.1f%% → %.1f%%", 
			cleaned, freedMB, usage, finalUsage)
	}

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
	Status  string
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
			Status:  "unknown",
		})
	}

	sort.Slice(jobs, func(i, j int) bool {
		return jobs[i].ModTime.Before(jobs[j].ModTime)
	})

	return jobs, nil
}

// getJobsForCleanup returns jobs sorted by cleanup priority
func (s *DiskService) getJobsForCleanup(isEmergency bool) ([]JobInfo, error) {
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

		jobDir := filepath.Join(s.config.Storage.OutputsDir, entry.Name())
		size, _ := utils.GetDirSize(jobDir)

		// Try to read state to get job status
		status := s.getJobStatus(entry.Name())

		jobs = append(jobs, JobInfo{
			JobID:   entry.Name(),
			ModTime: info.ModTime(),
			Size:    size,
			Status:  status,
		})
	}

	// Sort by priority for cleanup
	sort.Slice(jobs, func(i, j int) bool {
		// Priority order for cleanup:
		// 1. Failed jobs (highest priority)
		// 2. Interrupted jobs
		// 3. Completed jobs older than 7 days
		// 4. Oldest jobs (LRU)

		iPriority := s.getCleanupPriority(jobs[i], isEmergency)
		jPriority := s.getCleanupPriority(jobs[j], isEmergency)

		if iPriority != jPriority {
			return iPriority > jPriority // Higher priority first
		}

		// Same priority, use age (older first)
		return jobs[i].ModTime.Before(jobs[j].ModTime)
	})

	return jobs, nil
}

// getJobStatus reads job status from state file
func (s *DiskService) getJobStatus(jobID string) string {
	stateFile := filepath.Join(s.config.Storage.OutputsDir, jobID, "state.json")
	if !utils.FileExists(stateFile) {
		return "unknown"
	}

	data, err := os.ReadFile(stateFile)
	if err != nil {
		return "unknown"
	}

	var state struct {
		Status string `json:"status"`
	}

	if err := json.Unmarshal(data, &state); err != nil {
		return "unknown"
	}

	return state.Status
}

// getCleanupPriority returns cleanup priority (higher = clean first)
func (s *DiskService) getCleanupPriority(job JobInfo, isEmergency bool) int {
	age := time.Since(job.ModTime)

	switch job.Status {
	case "FAILED", "failed":
		return 100 // Highest priority - always clean failed jobs first

	case "INTERRUPTED", "interrupted":
		if age > 24*time.Hour {
			return 90 // Clean old interrupted jobs
		}
		return 50 // Keep recent interrupted jobs for recovery

	case "COMPLETED", "completed":
		if isEmergency {
			// In emergency, clean completed jobs older than 3 days
			if age > 3*24*time.Hour {
				return 80
			}
			return 40
		}
		// Normal cleanup: completed jobs older than 7 days
		if age > 7*24*time.Hour {
			return 70
		}
		return 30

	case "PARTIAL_SUCCESS", "partial_success":
		if age > 7*24*time.Hour {
			return 60
		}
		return 25

	default:
		// Unknown status - use age-based priority
		if age > 30*24*time.Hour {
			return 50
		}
		return 20
	}
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

// CheckJobSizeLimit checks if job size exceeds the maximum allowed
func (s *DiskService) CheckJobSizeLimit(jobID string) (bool, int64, error) {
	currentSize, err := s.GetJobSize(jobID)
	if err != nil {
		return false, 0, err
	}

	maxSizeMB := int64(s.config.Disk.MaxJobSizeMB)
	maxSizeBytes := maxSizeMB * 1024 * 1024
	currentSizeMB := currentSize / (1024 * 1024)

	if currentSize > maxSizeBytes {
		log.Printf("Job %s exceeded size limit: %d MB / %d MB", jobID, currentSizeMB, maxSizeMB)
		return false, currentSizeMB, nil
	}

	return true, currentSizeMB, nil
}

// EnforceJobSizeLimit enforces per-job disk limit and cleans up if exceeded
func (s *DiskService) EnforceJobSizeLimit(jobID string) error {
	withinLimit, currentSizeMB, err := s.CheckJobSizeLimit(jobID)
	if err != nil {
		return fmt.Errorf("failed to check job size: %w", err)
	}

	if !withinLimit {
		log.Printf("⚠️  Enforcing size limit for job %s: %d MB exceeds %d MB", 
			jobID, currentSizeMB, s.config.Disk.MaxJobSizeMB)
		
		// Delete job files
		if err := s.DeleteJobFiles(jobID); err != nil {
			log.Printf("Failed to delete job files for %s: %v", jobID, err)
		}

		return fmt.Errorf("job size limit exceeded: %d MB / %d MB", 
			currentSizeMB, s.config.Disk.MaxJobSizeMB)
	}

	return nil
}

// GetJobSizeMB returns job size in megabytes
func (s *DiskService) GetJobSizeMB(jobID string) (int64, error) {
	sizeBytes, err := s.GetJobSize(jobID)
	if err != nil {
		return 0, err
	}
	return sizeBytes / (1024 * 1024), nil
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
