package services

import (
	"encoder-service/internal/core/ports"
	"encoder-service/pkg/config"
	"encoder-service/pkg/types"
	"fmt"
	"time"
)

type JobService struct {
	repo   ports.JobRepository
	config *config.Config
}

func NewJobService(repo ports.JobRepository, cfg *config.Config) *JobService {
	return &JobService{
		repo:   repo,
		config: cfg,
	}
}

func (s *JobService) CreateJob(inputPath string, chunkDuration int) (*types.Job, error) {
	if chunkDuration == 0 {
		chunkDuration = s.config.Encoding.DefaultChunkDuration
	}

	now := time.Now()
	job := &types.Job{
		ID:            fmt.Sprintf("job_%d", now.UnixMilli()),
		Status:        types.StatusQueued,
		InputPath:     inputPath,
		ChunkDuration: chunkDuration,
		MaxRetries:    s.config.Worker.MaxRetries,
		Meta: types.Meta{
			CreatedAt: now,
		},
		Message: "Job created and queued for processing",
	}

	if err := s.repo.Create(job); err != nil {
		return nil, fmt.Errorf("failed to create job: %w", err)
	}

	return job, nil
}

func (s *JobService) GetJob(id string) (*types.Job, error) {
	return s.repo.Get(id)
}

func (s *JobService) ListJobs() ([]*types.Job, error) {
	return s.repo.List()
}

func (s *JobService) DeleteJob(id string) error {
	return s.repo.Delete(id)
}

func (s *JobService) UpdateStatus(id string, status types.JobStatus) error {
	job, err := s.repo.Get(id)
	if err != nil {
		return err
	}

	job.Status = status

	if status == types.StatusCompleted || status == types.StatusFailed {
		now := time.Now()
		job.Meta.CompletedAt = &now
		job.Meta.ProcessingTimeSec = now.Sub(job.Meta.CreatedAt).Seconds()
	}

	return s.repo.Update(job)
}

func (s *JobService) UpdateProgress(id string, percentage int, step string) error {
	job, err := s.repo.Get(id)
	if err != nil {
		return err
	}

	job.Progress = &types.Progress{
		Percentage:  percentage,
		CurrentStep: step,
	}

	return s.repo.Update(job)
}

func (s *JobService) SetError(id string, jobErr *types.JobError) error {
	job, err := s.repo.Get(id)
	if err != nil {
		return err
	}

	job.Error = jobErr
	job.Status = types.StatusFailed

	now := time.Now()
	job.Meta.CompletedAt = &now
	job.Meta.ProcessingTimeSec = now.Sub(job.Meta.CreatedAt).Seconds()

	return s.repo.Update(job)
}

func (s *JobService) SetMetadata(id string, input *types.VideoMetadata, output *types.OutputInfo) error {
	job, err := s.repo.Get(id)
	if err != nil {
		return err
	}

	job.Input = input
	job.Output = output

	return s.repo.Update(job)
}

func (s *JobService) SetHash(id, hash string) error {
	job, err := s.repo.Get(id)
	if err != nil {
		return err
	}

	job.InputHash = hash
	return s.repo.Update(job)
}

func (s *JobService) IncrementRetry(id string) error {
	job, err := s.repo.Get(id)
	if err != nil {
		return err
	}

	job.Retries++
	return s.repo.Update(job)
}
