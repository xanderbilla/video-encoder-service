package services

import (
	"encoder-service/pkg/config"
	"encoder-service/pkg/types"
	"fmt"
	"sync"
	"time"
)

// BatchService handles batch processing of multiple files
type BatchService struct {
	config  *config.Config
	batches map[string]*Batch
	mu      sync.RWMutex
}

// Batch represents a batch of jobs
type Batch struct {
	ID          string                 `json:"batchId"`
	Jobs        []*types.Job           `json:"jobs"`
	Status      BatchStatus            `json:"status"`
	CreatedAt   time.Time              `json:"createdAt"`
	CompletedAt *time.Time             `json:"completedAt,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// BatchStatus represents the status of a batch
type BatchStatus string

const (
	BatchStatusPending    BatchStatus = "PENDING"
	BatchStatusProcessing BatchStatus = "PROCESSING"
	BatchStatusCompleted  BatchStatus = "COMPLETED"
	BatchStatusFailed     BatchStatus = "FAILED"
	BatchStatusPartial    BatchStatus = "PARTIAL" // Some jobs succeeded, some failed
)

// BatchStats represents statistics for a batch
type BatchStats struct {
	Total     int `json:"total"`
	Pending   int `json:"pending"`
	Running   int `json:"running"`
	Completed int `json:"completed"`
	Failed    int `json:"failed"`
}

func NewBatchService(cfg *config.Config) *BatchService {
	return &BatchService{
		config:  cfg,
		batches: make(map[string]*Batch),
	}
}

// CreateBatch creates a new batch
func (s *BatchService) CreateBatch(batchID string, jobs []*types.Job, metadata map[string]interface{}) (*Batch, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.batches[batchID]; exists {
		return nil, fmt.Errorf("batch %s already exists", batchID)
	}

	batch := &Batch{
		ID:        batchID,
		Jobs:      jobs,
		Status:    BatchStatusPending,
		CreatedAt: time.Now(),
		Metadata:  metadata,
	}

	s.batches[batchID] = batch
	return batch, nil
}

// GetBatch retrieves a batch by ID
func (s *BatchService) GetBatch(batchID string) (*Batch, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	batch, exists := s.batches[batchID]
	if !exists {
		return nil, fmt.Errorf("batch %s not found", batchID)
	}

	return batch, nil
}

// UpdateBatchStatus updates the status of a batch
func (s *BatchService) UpdateBatchStatus(batchID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	batch, exists := s.batches[batchID]
	if !exists {
		return fmt.Errorf("batch %s not found", batchID)
	}

	stats := s.calculateBatchStats(batch)

	// Determine batch status based on job statuses
	if stats.Completed == stats.Total {
		batch.Status = BatchStatusCompleted
		now := time.Now()
		batch.CompletedAt = &now
	} else if stats.Failed == stats.Total {
		batch.Status = BatchStatusFailed
		now := time.Now()
		batch.CompletedAt = &now
	} else if stats.Completed > 0 && (stats.Completed+stats.Failed) == stats.Total {
		batch.Status = BatchStatusPartial
		now := time.Now()
		batch.CompletedAt = &now
	} else if stats.Running > 0 {
		batch.Status = BatchStatusProcessing
	} else {
		batch.Status = BatchStatusPending
	}

	return nil
}

// GetBatchStats returns statistics for a batch
func (s *BatchService) GetBatchStats(batchID string) (*BatchStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	batch, exists := s.batches[batchID]
	if !exists {
		return nil, fmt.Errorf("batch %s not found", batchID)
	}

	stats := s.calculateBatchStats(batch)
	return &stats, nil
}

// ListBatches returns all batches
func (s *BatchService) ListBatches() []*Batch {
	s.mu.RLock()
	defer s.mu.RUnlock()

	batches := make([]*Batch, 0, len(s.batches))
	for _, batch := range s.batches {
		batches = append(batches, batch)
	}
	return batches
}

// DeleteBatch deletes a batch
func (s *BatchService) DeleteBatch(batchID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.batches[batchID]; !exists {
		return fmt.Errorf("batch %s not found", batchID)
	}

	delete(s.batches, batchID)
	return nil
}

func (s *BatchService) calculateBatchStats(batch *Batch) BatchStats {
	stats := BatchStats{
		Total: len(batch.Jobs),
	}

	for _, job := range batch.Jobs {
		switch job.Status {
		case types.StatusQueued, types.JobStatusScheduled:
			stats.Pending++
		case types.StatusRunning:
			stats.Running++
		case types.StatusCompleted:
			stats.Completed++
		case types.StatusFailed:
			stats.Failed++
		}
	}

	return stats
}
