package handlers

import (
	"context"
	"encoding/json"
	"encoder-service/internal/application/services"
	"encoder-service/internal/application/usecases"
	"encoder-service/internal/core/domain"
	"encoder-service/internal/core/ports"
	"encoder-service/internal/infrastructure/filesystem"
	"encoder-service/internal/infrastructure/websocket"
	"encoder-service/pkg/types"
	"encoder-service/pkg/utils"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
)

type JobHandler struct {
	jobService       *services.JobService
	diskService      *services.DiskService
	validator        ports.FileValidator
	transcodeUseCase *usecases.TranscodeUseCase
	webhookService   *services.WebhookService
	schedulerService *services.SchedulerService
	batchService     *services.BatchService
	cdnService       *services.CDNService
	wsManager        *websocket.Manager
	jobQueue         chan *types.Job
	priorityQueue    *types.PriorityQueue
	maxWorkers       int
	activeWorkers    int
	mu               sync.RWMutex
}

func NewJobHandler(
	jobService *services.JobService,
	diskService *services.DiskService,
	validator ports.FileValidator,
	transcodeUseCase *usecases.TranscodeUseCase,
	webhookService *services.WebhookService,
	schedulerService *services.SchedulerService,
	batchService *services.BatchService,
	cdnService *services.CDNService,
	wsManager *websocket.Manager,
	jobQueue chan *types.Job,
	maxWorkers int,
) *JobHandler {
	return &JobHandler{
		jobService:       jobService,
		diskService:      diskService,
		validator:        validator,
		transcodeUseCase: transcodeUseCase,
		webhookService:   webhookService,
		schedulerService: schedulerService,
		batchService:     batchService,
		cdnService:       cdnService,
		wsManager:        wsManager,
		jobQueue:         jobQueue,
		priorityQueue:    types.NewPriorityQueue(),
		maxWorkers:       maxWorkers,
		activeWorkers:    0,
	}
}

type TranscodeRequest struct {
	InputPath      string                  `json:"inputPath"`
	ChunkDuration  int                     `json:"chunkDuration"`
	WebhookURL     string                  `json:"webhookUrl,omitempty"`
	WebhookConfig  *types.WebhookConfig    `json:"webhookConfig,omitempty"`
	ScheduledAt    *time.Time              `json:"scheduledAt,omitempty"`
	OutputFormats  []types.OutputFormat    `json:"outputFormats,omitempty"`
	PreviewConfig  *types.PreviewConfig    `json:"previewConfig,omitempty"`
	StorageConfig  *types.StorageConfig    `json:"storageConfig,omitempty"`
	CDNConfig      *types.CDNConfig        `json:"cdnConfig,omitempty"`
	Priority       types.JobPriority       `json:"priority,omitempty"`
}

func (h *JobHandler) CreateJob(w http.ResponseWriter, r *http.Request) {
	requestID := r.Header.Get("X-Request-ID")

	// Check disk space
	if canAccept, reason := h.diskService.CanAcceptNewJob(); !canAccept {
		utils.SendError(w, http.StatusServiceUnavailable, requestID, domain.CodeDiskFull, reason, nil)
		return
	}

	var req TranscodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.SendError(w, http.StatusBadRequest, requestID, domain.CodeInvalidRequest, "Invalid request body", map[string]interface{}{"error": err.Error()})
		return
	}

	if req.InputPath == "" {
		utils.SendError(w, http.StatusBadRequest, requestID, domain.CodeMissingField, "inputPath is required", nil)
		return
	}

	// Validate input file
	if err := h.validator.ValidateFile(req.InputPath); err != nil {
		if validationErr, ok := err.(*filesystem.ValidationError); ok {
			utils.SendError(w, http.StatusBadRequest, requestID, validationErr.Code, validationErr.Message, validationErr.Details)
		} else {
			utils.SendError(w, http.StatusBadRequest, requestID, domain.CodeInvalidVideo, "File validation failed", map[string]interface{}{"error": err.Error()})
		}
		return
	}

	// Create job
	job, err := h.jobService.CreateJob(req.InputPath, req.ChunkDuration)
	if err != nil {
		utils.SendError(w, http.StatusInternalServerError, requestID, domain.CodeInternalError, "Failed to create job", map[string]interface{}{"error": err.Error()})
		return
	}

	// Queue job for processing
	h.jobQueue <- job

	response := map[string]interface{}{
		"jobId":  job.ID,
		"status": string(job.Status),
	}

	utils.SendSuccessWithMessage(w, http.StatusOK, requestID, "Transcoding job created and queued successfully", response)
}

func (h *JobHandler) GetJob(w http.ResponseWriter, r *http.Request) {
	requestID := r.Header.Get("X-Request-ID")
	vars := mux.Vars(r)
	jobID := vars["id"]

	job, err := h.jobService.GetJob(jobID)
	if err != nil {
		utils.SendError(w, http.StatusNotFound, requestID, domain.CodeJobNotFound, "Job not found", map[string]interface{}{"jobId": jobID})
		return
	}

	utils.SendSuccess(w, http.StatusOK, requestID, job)
}

func (h *JobHandler) ListJobs(w http.ResponseWriter, r *http.Request) {
	requestID := r.Header.Get("X-Request-ID")

	jobs, err := h.jobService.ListJobs()
	if err != nil {
		utils.SendError(w, http.StatusInternalServerError, requestID, domain.CodeInternalError, "Failed to list jobs", map[string]interface{}{"error": err.Error()})
		return
	}

	response := map[string]interface{}{
		"jobs":  jobs,
		"count": len(jobs),
	}

	utils.SendSuccess(w, http.StatusOK, requestID, response)
}

func (h *JobHandler) DeleteJob(w http.ResponseWriter, r *http.Request) {
	requestID := r.Header.Get("X-Request-ID")
	vars := mux.Vars(r)
	jobID := vars["id"]

	_, err := h.jobService.GetJob(jobID)
	if err != nil {
		utils.SendError(w, http.StatusNotFound, requestID, domain.CodeJobNotFound, "Job not found", map[string]interface{}{"jobId": jobID})
		return
	}

	// Delete files
	if err := h.diskService.DeleteJobFiles(jobID); err != nil {
		utils.SendError(w, http.StatusInternalServerError, requestID, domain.CodeDeleteFailed, "Failed to delete output files", map[string]interface{}{"error": err.Error()})
		return
	}

	// Delete job from repository
	if err := h.jobService.DeleteJob(jobID); err != nil {
		utils.SendError(w, http.StatusInternalServerError, requestID, domain.CodeDeleteFailed, "Failed to delete job", map[string]interface{}{"error": err.Error()})
		return
	}

	response := map[string]interface{}{
		"jobId":   jobID,
		"deleted": true,
	}

	utils.SendSuccessWithMessage(w, http.StatusOK, requestID, "Job and all associated files deleted successfully", response)
}

func (h *JobHandler) ResumeJob(w http.ResponseWriter, r *http.Request) {
	requestID := r.Header.Get("X-Request-ID")
	vars := mux.Vars(r)
	jobID := vars["id"]

	job, err := h.jobService.GetJob(jobID)
	if err != nil {
		utils.SendError(w, http.StatusNotFound, requestID, domain.CodeJobNotFound, "Job not found", map[string]interface{}{"jobId": jobID})
		return
	}

	// Check if job can be resumed
	if job.Status != types.StatusFailed && job.Status != types.StatusInterrupted {
		utils.SendError(w, http.StatusBadRequest, requestID, domain.CodeInvalidRequest, 
			"Job cannot be resumed. Only FAILED or INTERRUPTED jobs can be resumed.", 
			map[string]interface{}{"currentStatus": string(job.Status)})
		return
	}

	// Reset status and requeue
	h.jobService.UpdateStatus(jobID, types.StatusQueued)
	h.jobQueue <- job

	response := map[string]interface{}{
		"jobId":   jobID,
		"resumed": true,
		"status":  "QUEUED",
	}

	utils.SendSuccessWithMessage(w, http.StatusOK, requestID, "Job resumed successfully", response)
}

func (h *JobHandler) Health(w http.ResponseWriter, r *http.Request) {
	requestID := r.Header.Get("X-Request-ID")

	// Get disk usage
	diskUsage, _ := utils.GetDiskUsagePercent(".")
	total, used, free, _ := utils.GetDiskSpace(".")

	// Get job statistics
	jobs, _ := h.jobService.ListJobs()
	queuedCount := 0
	runningCount := 0
	completedCount := 0
	failedCount := 0

	for _, job := range jobs {
		switch job.Status {
		case types.StatusQueued:
			queuedCount++
		case types.StatusRunning:
			runningCount++
		case types.StatusCompleted:
			completedCount++
		case types.StatusFailed:
			failedCount++
		}
	}

	h.mu.RLock()
	activeWorkers := h.activeWorkers
	h.mu.RUnlock()

	response := map[string]interface{}{
		"status":  "ok",
		"service": "video-encoder",
		"version": "2.0.0",
		"uptime":  time.Since(startTime).String(),
		"workers": map[string]interface{}{
			"active": activeWorkers,
			"max":    h.maxWorkers,
		},
		"queue": map[string]interface{}{
			"size":     len(h.jobQueue),
			"capacity": cap(h.jobQueue),
		},
		"jobs": map[string]interface{}{
			"total":     len(jobs),
			"queued":    queuedCount,
			"running":   runningCount,
			"completed": completedCount,
			"failed":    failedCount,
		},
		"disk": map[string]interface{}{
			"usagePercent": float64(int(diskUsage*100)) / 100,
			"totalGB":      float64(total) / (1024 * 1024 * 1024),
			"usedGB":       float64(used) / (1024 * 1024 * 1024),
			"freeGB":       float64(free) / (1024 * 1024 * 1024),
		},
	}

	utils.SendSuccessWithMessage(w, http.StatusOK, requestID, "Service is healthy and running", response)
}

var startTime = time.Now()

// StartWorker processes jobs from the queue
func (h *JobHandler) StartWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case job := <-h.jobQueue:
			h.mu.Lock()
			h.activeWorkers++
			h.mu.Unlock()

			go func(j *types.Job) {
				defer func() {
					h.mu.Lock()
					h.activeWorkers--
					h.mu.Unlock()
				}()
				h.transcodeUseCase.Execute(ctx, j)
			}(job)
		}
	}
}

// HandleWebSocket handles WebSocket connections for real-time progress updates
func (h *JobHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := vars["id"]

	// Verify job exists
	if _, err := h.jobService.GetJob(jobID); err != nil {
		http.Error(w, "Job not found", http.StatusNotFound)
		return
	}

	// Handle WebSocket connection
	h.wsManager.HandleConnection(w, r, jobID)
}

// CreateBatchJob creates a batch of transcoding jobs
func (h *JobHandler) CreateBatchJob(w http.ResponseWriter, r *http.Request) {
	requestID := r.Header.Get("X-Request-ID")

	var req struct {
		Files         []string                `json:"files"`
		BatchID       string                  `json:"batchId,omitempty"`
		ChunkDuration int                     `json:"chunkDuration,omitempty"`
		WebhookURL    string                  `json:"webhookUrl,omitempty"`
		Metadata      map[string]interface{}  `json:"metadata,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.SendError(w, http.StatusBadRequest, requestID, domain.CodeInvalidRequest, "Invalid request body", map[string]interface{}{"error": err.Error()})
		return
	}

	if len(req.Files) == 0 {
		utils.SendError(w, http.StatusBadRequest, requestID, domain.CodeMissingField, "files array is required", nil)
		return
	}

	// Generate batch ID if not provided
	if req.BatchID == "" {
		req.BatchID = "batch_" + time.Now().Format("20060102150405")
	}

	// Create jobs for each file
	jobs := make([]*types.Job, 0, len(req.Files))
	for _, filePath := range req.Files {
		// Validate file
		if err := h.validator.ValidateFile(filePath); err != nil {
			utils.SendError(w, http.StatusBadRequest, requestID, domain.CodeInvalidVideo, "File validation failed for "+filePath, map[string]interface{}{"error": err.Error()})
			return
		}

		// Create job
		job, err := h.jobService.CreateJob(filePath, req.ChunkDuration)
		if err != nil {
			utils.SendError(w, http.StatusInternalServerError, requestID, domain.CodeInternalError, "Failed to create job for "+filePath, map[string]interface{}{"error": err.Error()})
			return
		}

		jobs = append(jobs, job)
		h.jobQueue <- job
	}

	// Create batch
	batch, err := h.batchService.CreateBatch(req.BatchID, jobs, req.Metadata)
	if err != nil {
		utils.SendError(w, http.StatusInternalServerError, requestID, domain.CodeInternalError, "Failed to create batch", map[string]interface{}{"error": err.Error()})
		return
	}

	response := map[string]interface{}{
		"batchId": batch.ID,
		"jobs":    jobs,
		"status":  string(batch.Status),
	}

	utils.SendSuccessWithMessage(w, http.StatusOK, requestID, "Batch job created successfully", response)
}

// GetBatchStatus returns the status of a batch job
func (h *JobHandler) GetBatchStatus(w http.ResponseWriter, r *http.Request) {
	requestID := r.Header.Get("X-Request-ID")
	vars := mux.Vars(r)
	batchID := vars["id"]

	batch, err := h.batchService.GetBatch(batchID)
	if err != nil {
		utils.SendError(w, http.StatusNotFound, requestID, domain.CodeJobNotFound, "Batch not found", nil)
		return
	}

	// Update batch status
	h.batchService.UpdateBatchStatus(batchID)

	// Get batch stats
	stats, _ := h.batchService.GetBatchStats(batchID)

	response := map[string]interface{}{
		"batchId":     batch.ID,
		"status":      string(batch.Status),
		"createdAt":   batch.CreatedAt,
		"completedAt": batch.CompletedAt,
		"stats":       stats,
		"jobs":        batch.Jobs,
	}

	utils.SendSuccessWithMessage(w, http.StatusOK, requestID, "Batch status retrieved successfully", response)
}

// ScheduleJob schedules a job for future execution
func (h *JobHandler) ScheduleJob(w http.ResponseWriter, r *http.Request) {
	requestID := r.Header.Get("X-Request-ID")

	var req TranscodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.SendError(w, http.StatusBadRequest, requestID, domain.CodeInvalidRequest, "Invalid request body", map[string]interface{}{"error": err.Error()})
		return
	}

	if req.InputPath == "" {
		utils.SendError(w, http.StatusBadRequest, requestID, domain.CodeMissingField, "inputPath is required", nil)
		return
	}

	if req.ScheduledAt == nil {
		utils.SendError(w, http.StatusBadRequest, requestID, domain.CodeMissingField, "scheduledAt is required", nil)
		return
	}

	// Validate input file
	if err := h.validator.ValidateFile(req.InputPath); err != nil {
		utils.SendError(w, http.StatusBadRequest, requestID, domain.CodeInvalidVideo, "File validation failed", map[string]interface{}{"error": err.Error()})
		return
	}

	// Create job
	job, err := h.jobService.CreateJob(req.InputPath, req.ChunkDuration)
	if err != nil {
		utils.SendError(w, http.StatusInternalServerError, requestID, domain.CodeInternalError, "Failed to create job", map[string]interface{}{"error": err.Error()})
		return
	}

	// Set job status to scheduled
	job.Status = types.JobStatusScheduled

	// Schedule job
	if err := h.schedulerService.ScheduleJob(job, *req.ScheduledAt); err != nil {
		utils.SendError(w, http.StatusInternalServerError, requestID, domain.CodeInternalError, "Failed to schedule job", map[string]interface{}{"error": err.Error()})
		return
	}

	response := map[string]interface{}{
		"jobId":       job.ID,
		"status":      string(job.Status),
		"scheduledAt": req.ScheduledAt,
	}

	utils.SendSuccessWithMessage(w, http.StatusOK, requestID, "Job scheduled successfully", response)
}

// GetDetailedVideoInfo returns comprehensive video metadata
func (h *JobHandler) GetDetailedVideoInfo(w http.ResponseWriter, r *http.Request) {
	requestID := r.Header.Get("X-Request-ID")

	var req struct {
		InputPath string `json:"inputPath"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.SendError(w, http.StatusBadRequest, requestID, domain.CodeInvalidRequest, "Invalid request body", map[string]interface{}{"error": err.Error()})
		return
	}

	if req.InputPath == "" {
		utils.SendError(w, http.StatusBadRequest, requestID, domain.CodeMissingField, "inputPath is required", nil)
		return
	}

	// Validate file exists
	if err := h.validator.ValidateFile(req.InputPath); err != nil {
		utils.SendError(w, http.StatusBadRequest, requestID, domain.CodeInvalidVideo, "File validation failed", map[string]interface{}{"error": err.Error()})
		return
	}

	// Get detailed video info
	// This would require adding the probe to the handler or use case
	// For now, return a placeholder response
	response := map[string]interface{}{
		"inputPath": req.InputPath,
		"message":   "Detailed video analysis endpoint - implementation pending",
	}

	utils.SendSuccessWithMessage(w, http.StatusOK, requestID, "Video info retrieved", response)
}
