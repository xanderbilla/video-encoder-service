package filesystem

import (
	"encoder-service/internal/core/domain"
	"encoder-service/pkg/types"
	"sync"
)

type InMemoryJobRepository struct {
	jobs map[string]*types.Job
	mu   sync.RWMutex
}

func NewInMemoryJobRepository() *InMemoryJobRepository {
	return &InMemoryJobRepository{
		jobs: make(map[string]*types.Job),
	}
}

func (r *InMemoryJobRepository) Create(job *types.Job) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.jobs[job.ID]; exists {
		return domain.ErrJobAlreadyExists
	}

	r.jobs[job.ID] = job
	return nil
}

func (r *InMemoryJobRepository) Get(id string) (*types.Job, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	job, exists := r.jobs[id]
	if !exists {
		return nil, domain.ErrJobNotFound
	}

	return job, nil
}

func (r *InMemoryJobRepository) Update(job *types.Job) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.jobs[job.ID]; !exists {
		return domain.ErrJobNotFound
	}

	r.jobs[job.ID] = job
	return nil
}

func (r *InMemoryJobRepository) Delete(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.jobs[id]; !exists {
		return domain.ErrJobNotFound
	}

	delete(r.jobs, id)
	return nil
}

func (r *InMemoryJobRepository) List() ([]*types.Job, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	jobList := make([]*types.Job, 0, len(r.jobs))
	for _, job := range r.jobs {
		jobList = append(jobList, job)
	}

	return jobList, nil
}

func (r *InMemoryJobRepository) Exists(id string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.jobs[id]
	return exists
}
