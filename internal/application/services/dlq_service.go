package services

import (
	"encoder-service/pkg/config"
	"encoder-service/pkg/types"
	"encoder-service/pkg/utils"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

// DLQService manages the Dead Letter Queue for permanently failed jobs
type DLQService struct {
	config *config.Config
	dlqDir string
}

func NewDLQService(cfg *config.Config) *DLQService {
	dlqDir := filepath.Join(cfg.Storage.OutputsDir, "dlq")
	utils.EnsureDir(dlqDir)
	
	return &DLQService{
		config: cfg,
		dlqDir: dlqDir,
	}
}

// MoveToDLQ moves a permanently failed job to the dead letter queue
func (s *DLQService) MoveToDLQ(job *types.Job, reason string) error {
	log.Printf("Moving job %s to DLQ: %s", job.ID, reason)

	// Create DLQ entry
	dlqEntry := DLQEntry{
		JobID:       job.ID,
		InputPath:   job.InputPath,
		Status:      string(job.Status),
		Error:       job.Error,
		Retries:     job.Retries,
		MaxRetries:  job.MaxRetries,
		Reason:      reason,
		MovedAt:     time.Now(),
		OriginalJob: job,
	}

	// Save to DLQ
	dlqPath := filepath.Join(s.dlqDir, fmt.Sprintf("%s.json", job.ID))
	data, err := json.MarshalIndent(dlqEntry, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal DLQ entry: %w", err)
	}

	if err := os.WriteFile(dlqPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write DLQ entry: %w", err)
	}

	log.Printf("Job %s moved to DLQ successfully", job.ID)
	return nil
}

// ListDLQJobs lists all jobs in the DLQ
func (s *DLQService) ListDLQJobs() ([]DLQEntry, error) {
	var entries []DLQEntry

	files, err := filepath.Glob(filepath.Join(s.dlqDir, "*.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to list DLQ files: %w", err)
	}

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			log.Printf("Warning: Failed to read DLQ file %s: %v", file, err)
			continue
		}

		var entry DLQEntry
		if err := json.Unmarshal(data, &entry); err != nil {
			log.Printf("Warning: Failed to unmarshal DLQ file %s: %v", file, err)
			continue
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

// GetDLQEntry retrieves a specific DLQ entry
func (s *DLQService) GetDLQEntry(jobID string) (*DLQEntry, error) {
	dlqPath := filepath.Join(s.dlqDir, fmt.Sprintf("%s.json", jobID))
	
	if !utils.FileExists(dlqPath) {
		return nil, fmt.Errorf("DLQ entry not found for job %s", jobID)
	}

	data, err := os.ReadFile(dlqPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read DLQ entry: %w", err)
	}

	var entry DLQEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, fmt.Errorf("failed to unmarshal DLQ entry: %w", err)
	}

	return &entry, nil
}

// RemoveFromDLQ removes a job from the DLQ
func (s *DLQService) RemoveFromDLQ(jobID string) error {
	dlqPath := filepath.Join(s.dlqDir, fmt.Sprintf("%s.json", jobID))
	
	if !utils.FileExists(dlqPath) {
		return fmt.Errorf("DLQ entry not found for job %s", jobID)
	}

	if err := os.Remove(dlqPath); err != nil {
		return fmt.Errorf("failed to remove DLQ entry: %w", err)
	}

	log.Printf("Job %s removed from DLQ", jobID)
	return nil
}

// CleanupOldDLQEntries removes DLQ entries older than specified days
func (s *DLQService) CleanupOldDLQEntries(daysOld int) error {
	entries, err := s.ListDLQJobs()
	if err != nil {
		return err
	}

	cutoff := time.Now().AddDate(0, 0, -daysOld)
	cleaned := 0

	for _, entry := range entries {
		if entry.MovedAt.Before(cutoff) {
			if err := s.RemoveFromDLQ(entry.JobID); err != nil {
				log.Printf("Warning: Failed to remove old DLQ entry %s: %v", entry.JobID, err)
				continue
			}
			cleaned++
		}
	}

	if cleaned > 0 {
		log.Printf("Cleaned up %d old DLQ entries", cleaned)
	}

	return nil
}

// DLQEntry represents a job in the dead letter queue
type DLQEntry struct {
	JobID       string          `json:"jobId"`
	InputPath   string          `json:"inputPath"`
	Status      string          `json:"status"`
	Error       *types.JobError `json:"error,omitempty"`
	Retries     int             `json:"retries"`
	MaxRetries  int             `json:"maxRetries"`
	Reason      string          `json:"reason"`
	MovedAt     time.Time       `json:"movedAt"`
	OriginalJob *types.Job      `json:"originalJob,omitempty"`
}
