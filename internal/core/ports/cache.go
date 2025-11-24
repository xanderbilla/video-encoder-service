package ports

// DeduplicationCache defines the interface for caching and deduplication
type DeduplicationCache interface {
	ComputeHash(filePath string) (string, error)
	CheckDuplicate(hash string) (string, bool)
	Store(hash, jobID string) error
	Delete(hash string) error
}
