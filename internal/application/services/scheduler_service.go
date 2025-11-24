package services

import (
	"context"
	"encoder-service/pkg/config"
	"encoder-service/pkg/types"
	"log"
	"sync"
	"time"
)

// SchedulerService handles delayed job execution
type SchedulerService struct {
	config         *config.Config
	scheduledJobs  map[string]*ScheduledJob
	mu             sync.RWMutex
	jobQueue       chan *types.Job
	stopChan       chan struct{}
	wg             sync.WaitGroup
}

// ScheduledJob represents a job scheduled for future execution
type ScheduledJob struct {
	Job         *types.Job
	ScheduledAt time.Time
	Timer       *time.Timer
}

func NewSchedulerService(cfg *config.Config, jobQueue chan *types.Job) *SchedulerService {
	return &SchedulerService{
		config:        cfg,
		scheduledJobs: make(map[string]*ScheduledJob),
		jobQueue:      jobQueue,
		stopChan:      make(chan struct{}),
	}
}

// ScheduleJob schedules a job for execution at a specific time
func (s *SchedulerService) ScheduleJob(job *types.Job, scheduledAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Calculate delay
	delay := time.Until(scheduledAt)
	if delay < 0 {
		// If scheduled time is in the past, execute immediately
		log.Printf("Job %s scheduled time is in the past, executing immediately", job.ID)
		s.jobQueue <- job
		return nil
	}

	// Create timer
	timer := time.AfterFunc(delay, func() {
		s.executeScheduledJob(job.ID)
	})

	// Store scheduled job
	s.scheduledJobs[job.ID] = &ScheduledJob{
		Job:         job,
		ScheduledAt: scheduledAt,
		Timer:       timer,
	}

	log.Printf("Job %s scheduled for execution at %s (in %v)", job.ID, scheduledAt.Format(time.RFC3339), delay)
	return nil
}

// CancelScheduledJob cancels a scheduled job
func (s *SchedulerService) CancelScheduledJob(jobID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	scheduledJob, exists := s.scheduledJobs[jobID]
	if !exists {
		return nil
	}

	// Stop timer
	scheduledJob.Timer.Stop()
	delete(s.scheduledJobs, jobID)

	log.Printf("Cancelled scheduled job %s", jobID)
	return nil
}

// GetScheduledJob returns a scheduled job by ID
func (s *SchedulerService) GetScheduledJob(jobID string) (*ScheduledJob, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	job, exists := s.scheduledJobs[jobID]
	return job, exists
}

// ListScheduledJobs returns all scheduled jobs
func (s *SchedulerService) ListScheduledJobs() []*ScheduledJob {
	s.mu.RLock()
	defer s.mu.RUnlock()

	jobs := make([]*ScheduledJob, 0, len(s.scheduledJobs))
	for _, job := range s.scheduledJobs {
		jobs = append(jobs, job)
	}
	return jobs
}

// Start starts the scheduler service
func (s *SchedulerService) Start(ctx context.Context) {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		<-ctx.Done()
		s.Stop()
	}()

	log.Println("Scheduler service started")
}

// Stop stops the scheduler service
func (s *SchedulerService) Stop() {
	close(s.stopChan)

	// Cancel all scheduled jobs
	s.mu.Lock()
	for jobID, scheduledJob := range s.scheduledJobs {
		scheduledJob.Timer.Stop()
		delete(s.scheduledJobs, jobID)
	}
	s.mu.Unlock()

	s.wg.Wait()
	log.Println("Scheduler service stopped")
}

func (s *SchedulerService) executeScheduledJob(jobID string) {
	s.mu.Lock()
	scheduledJob, exists := s.scheduledJobs[jobID]
	if !exists {
		s.mu.Unlock()
		return
	}

	job := scheduledJob.Job
	delete(s.scheduledJobs, jobID)
	s.mu.Unlock()

	// Update job status
	job.Status = types.StatusQueued

	// Add to job queue
	log.Printf("Executing scheduled job %s", jobID)
	s.jobQueue <- job
}
