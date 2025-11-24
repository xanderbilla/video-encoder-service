package services

import (
	"encoder-service/pkg/config"
	"encoder-service/pkg/utils"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

// MetricsService tracks resource usage and performance metrics
type MetricsService struct {
	config     *config.Config
	metricsDir string
	mu         sync.RWMutex
	jobMetrics map[string]*JobMetrics
}

// JobMetrics tracks metrics for a specific job
type JobMetrics struct {
	JobID              string        `json:"jobId"`
	StartTime          time.Time     `json:"startTime"`
	EndTime            *time.Time    `json:"endTime,omitempty"`
	Duration           time.Duration `json:"duration"`
	CPUUsagePercent    float64       `json:"cpuUsagePercent"`
	MemoryUsageMB      float64       `json:"memoryUsageMB"`
	PeakMemoryMB       float64       `json:"peakMemoryMB"`
	DiskUsedMB         float64       `json:"diskUsedMB"`
	InputSizeMB        float64       `json:"inputSizeMB"`
	OutputSizeMB       float64       `json:"outputSizeMB"`
	QualitiesGenerated int           `json:"qualitiesGenerated"`
	ChunksGenerated    int           `json:"chunksGenerated"`
	Status             string        `json:"status"`
}

// WorkerMetrics tracks overall worker performance
type WorkerMetrics struct {
	WorkerID        string    `json:"workerId"`
	LastUpdated     time.Time `json:"lastUpdated"`
	JobsProcessed   int       `json:"jobsProcessed"`
	JobsSucceeded   int       `json:"jobsSucceeded"`
	JobsFailed      int       `json:"jobsFailed"`
	TotalCPUPercent float64   `json:"totalCpuPercent"`
	TotalMemoryMB   float64   `json:"totalMemoryMB"`
	Uptime          string    `json:"uptime"`
}

func NewMetricsService(cfg *config.Config) *MetricsService {
	metricsDir := filepath.Join(cfg.Storage.LogsDir, "metrics")
	utils.EnsureDir(metricsDir)

	return &MetricsService{
		config:     cfg,
		metricsDir: metricsDir,
		jobMetrics: make(map[string]*JobMetrics),
	}
}

// StartJobMetrics initializes metrics tracking for a job
func (s *MetricsService) StartJobMetrics(jobID, inputPath string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	inputSize, _ := utils.GetFileSizeMB(inputPath)

	metrics := &JobMetrics{
		JobID:        jobID,
		StartTime:    time.Now(),
		InputSizeMB:  inputSize,
		Status:       "running",
	}

	s.jobMetrics[jobID] = metrics
}

// UpdateJobMetrics updates metrics during job processing
func (s *MetricsService) UpdateJobMetrics(jobID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	metrics, exists := s.jobMetrics[jobID]
	if !exists {
		return
	}

	// Get current memory stats
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	currentMemMB := float64(m.Alloc) / 1024 / 1024

	metrics.MemoryUsageMB = currentMemMB
	if currentMemMB > metrics.PeakMemoryMB {
		metrics.PeakMemoryMB = currentMemMB
	}

	// Update CPU usage (simplified - in production use proper CPU monitoring)
	metrics.CPUUsagePercent = float64(runtime.NumGoroutine()) / float64(runtime.NumCPU()) * 10
}

// EndJobMetrics finalizes metrics for a completed job
func (s *MetricsService) EndJobMetrics(jobID, status string, qualitiesGenerated, chunksGenerated int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	metrics, exists := s.jobMetrics[jobID]
	if !exists {
		return fmt.Errorf("metrics not found for job %s", jobID)
	}

	now := time.Now()
	metrics.EndTime = &now
	metrics.Duration = now.Sub(metrics.StartTime)
	metrics.Status = status
	metrics.QualitiesGenerated = qualitiesGenerated
	metrics.ChunksGenerated = chunksGenerated

	// Calculate output size
	outputDir := filepath.Join(s.config.Storage.OutputsDir, jobID)
	if outputSize, err := utils.GetDirSize(outputDir); err == nil {
		metrics.OutputSizeMB = float64(outputSize) / 1024 / 1024
	}

	// Calculate disk used
	metrics.DiskUsedMB = metrics.OutputSizeMB

	// Save metrics to file
	return s.saveMetrics(jobID, metrics)
}

// GetJobMetrics retrieves metrics for a job
func (s *MetricsService) GetJobMetrics(jobID string) (*JobMetrics, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	metrics, exists := s.jobMetrics[jobID]
	if !exists {
		// Try to load from file
		return s.loadMetrics(jobID)
	}

	return metrics, nil
}

// SaveWorkerMetrics saves worker-level metrics
func (s *MetricsService) SaveWorkerMetrics(workerID string, metrics *WorkerMetrics) error {
	metricsPath := filepath.Join(s.metricsDir, fmt.Sprintf("worker_%s.json", workerID))
	
	data, err := json.MarshalIndent(metrics, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal worker metrics: %w", err)
	}

	if err := os.WriteFile(metricsPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write worker metrics: %w", err)
	}

	return nil
}

func (s *MetricsService) saveMetrics(jobID string, metrics *JobMetrics) error {
	metricsPath := filepath.Join(s.metricsDir, fmt.Sprintf("job_%s.json", jobID))
	
	data, err := json.MarshalIndent(metrics, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metrics: %w", err)
	}

	if err := os.WriteFile(metricsPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write metrics: %w", err)
	}

	log.Printf("Saved metrics for job %s", jobID)
	return nil
}

func (s *MetricsService) loadMetrics(jobID string) (*JobMetrics, error) {
	metricsPath := filepath.Join(s.metricsDir, fmt.Sprintf("job_%s.json", jobID))
	
	if !utils.FileExists(metricsPath) {
		return nil, fmt.Errorf("metrics file not found for job %s", jobID)
	}

	data, err := os.ReadFile(metricsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read metrics: %w", err)
	}

	var metrics JobMetrics
	if err := json.Unmarshal(data, &metrics); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metrics: %w", err)
	}

	return &metrics, nil
}

// StartMetricsMonitoring starts periodic metrics collection for a job
func (s *MetricsService) StartMetricsMonitoring(jobID string, stop chan bool) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.UpdateJobMetrics(jobID)
		case <-stop:
			return
		}
	}
}
