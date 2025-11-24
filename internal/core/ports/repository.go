package ports

import "encoder-service/pkg/types"

// JobRepository defines the interface for job storage
type JobRepository interface {
	Create(job *types.Job) error
	Get(id string) (*types.Job, error)
	Update(job *types.Job) error
	Delete(id string) error
	List() ([]*types.Job, error)
	Exists(id string) bool
}
