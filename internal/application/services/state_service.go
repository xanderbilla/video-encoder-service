package services

import (
	"encoder-service/pkg/config"
	"encoder-service/pkg/types"
	"encoder-service/pkg/utils"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type StateService struct {
	config *config.Config
	mu     sync.RWMutex
}

func NewStateService(cfg *config.Config) *StateService {
	return &StateService{
		config: cfg,
	}
}

// CreateState creates a new job state
func (s *StateService) CreateState(jobID, inputPath string, chunkDuration int) (*types.JobState, error) {
	now := time.Now()
	state := &types.JobState{
		JobID:             jobID,
		Status:            "processing",
		CurrentStep:       string(types.StepValidation),
		StepsCompleted:    []string{},
		QualitiesComplete: []string{},
		QualitiesFailed:   []string{},
		ChunksCompleted:   0,
		TotalChunks:       0,
		LastHeartbeat:     now,
		CreatedAt:         now,
		UpdatedAt:         now,
		ResumeSupported:   true,
		InputPath:         inputPath,
		ChunkDuration:     chunkDuration,
	}

	if err := s.SaveState(state); err != nil {
		return nil, fmt.Errorf("failed to save initial state: %w", err)
	}

	return state, nil
}

// LoadState loads job state from disk
func (s *StateService) LoadState(jobID string) (*types.JobState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	statePath := s.getStatePath(jobID)
	
	if !utils.FileExists(statePath) {
		return nil, fmt.Errorf("state file not found for job %s", jobID)
	}

	data, err := os.ReadFile(statePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	var state types.JobState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state: %w", err)
	}

	return &state, nil
}

// SaveState saves job state to disk atomically
func (s *StateService) SaveState(state *types.JobState) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	state.UpdatedAt = time.Now()

	// Ensure state directory exists
	stateDir := filepath.Join(s.config.Storage.OutputsDir, state.JobID)
	if err := utils.EnsureDir(stateDir); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	statePath := s.getStatePath(state.JobID)
	tempPath := statePath + ".tmp"

	// Marshal state to JSON
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	// Write to temp file first (atomic operation)
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp state file: %w", err)
	}

	// Rename temp file to actual file (atomic operation)
	if err := os.Rename(tempPath, statePath); err != nil {
		return fmt.Errorf("failed to rename state file: %w", err)
	}

	return nil
}

// UpdateState updates specific fields in the state
func (s *StateService) UpdateState(jobID string, updateFn func(*types.JobState)) error {
	state, err := s.LoadState(jobID)
	if err != nil {
		return err
	}

	updateFn(state)

	return s.SaveState(state)
}

// DeleteState removes the state file
func (s *StateService) DeleteState(jobID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	statePath := s.getStatePath(jobID)
	if utils.FileExists(statePath) {
		return os.Remove(statePath)
	}
	return nil
}

// FindOrphanedJobs finds all jobs that are orphaned
func (s *StateService) FindOrphanedJobs() ([]*types.JobState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var orphaned []*types.JobState

	// Scan outputs directory for state files
	entries, err := os.ReadDir(s.config.Storage.OutputsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read outputs directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		jobID := entry.Name()
		statePath := s.getStatePath(jobID)

		if !utils.FileExists(statePath) {
			continue
		}

		// Load state
		data, err := os.ReadFile(statePath)
		if err != nil {
			continue
		}

		var state types.JobState
		if err := json.Unmarshal(data, &state); err != nil {
			continue
		}

		// Check if orphaned
		if state.Status == "processing" && state.IsOrphaned() {
			orphaned = append(orphaned, &state)
		}
	}

	return orphaned, nil
}

// UpdateHeartbeat updates the heartbeat for a job
func (s *StateService) UpdateHeartbeat(jobID string) error {
	return s.UpdateState(jobID, func(state *types.JobState) {
		state.UpdateHeartbeat()
	})
}

func (s *StateService) getStatePath(jobID string) string {
	return filepath.Join(s.config.Storage.OutputsDir, jobID, "state.json")
}
