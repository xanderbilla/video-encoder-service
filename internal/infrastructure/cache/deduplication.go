package cache

import (
	"encoder-service/pkg/utils"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type DeduplicationCache struct {
	cacheDir string
	hashMap  map[string]string // hash -> jobID
	mu       sync.RWMutex
}

func NewDeduplicationCache(cacheDir string) *DeduplicationCache {
	cache := &DeduplicationCache{
		cacheDir: cacheDir,
		hashMap:  make(map[string]string),
	}

	// Ensure cache directory exists
	utils.EnsureDir(cacheDir)

	// Load existing cache
	cache.load()

	return cache
}

func (c *DeduplicationCache) ComputeHash(filePath string) (string, error) {
	return utils.ComputeFileHash(filePath)
}

func (c *DeduplicationCache) CheckDuplicate(hash string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	jobID, exists := c.hashMap[hash]
	return jobID, exists
}

func (c *DeduplicationCache) Store(hash, jobID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.hashMap[hash] = jobID
	return c.save()
}

func (c *DeduplicationCache) Delete(hash string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.hashMap, hash)
	return c.save()
}

func (c *DeduplicationCache) load() error {
	cacheFile := filepath.Join(c.cacheDir, "dedup.json")

	if !utils.FileExists(cacheFile) {
		return nil
	}

	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return fmt.Errorf("failed to read cache file: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if err := json.Unmarshal(data, &c.hashMap); err != nil {
		return fmt.Errorf("failed to unmarshal cache: %w", err)
	}

	return nil
}

func (c *DeduplicationCache) save() error {
	cacheFile := filepath.Join(c.cacheDir, "dedup.json")

	data, err := json.MarshalIndent(c.hashMap, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache: %w", err)
	}

	if err := os.WriteFile(cacheFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	return nil
}
