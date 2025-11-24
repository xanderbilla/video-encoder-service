package usecases

import (
	"context"
	"encoder-service/internal/application/services"
	"encoder-service/internal/core/domain"
	"encoder-service/internal/core/ports"
	"encoder-service/internal/infrastructure/ffmpeg"
	"encoder-service/pkg/types"
	"encoder-service/pkg/utils"
	"fmt"
	"log"
	"path/filepath"
	"time"
)

type TranscodeUseCase struct {
	jobService     *services.JobService
	diskService    *services.DiskService
	stateService   *services.StateService
	dlqService     *services.DLQService
	metricsService *services.MetricsService
	encoder        ports.VideoEncoder
	validator      ports.FileValidator
	cache          ports.DeduplicationCache
	maxRetries     int
}

func NewTranscodeUseCase(
	jobService *services.JobService,
	diskService *services.DiskService,
	stateService *services.StateService,
	dlqService *services.DLQService,
	metricsService *services.MetricsService,
	encoder ports.VideoEncoder,
	validator ports.FileValidator,
	cache ports.DeduplicationCache,
	maxRetries int,
) *TranscodeUseCase {
	return &TranscodeUseCase{
		jobService:     jobService,
		diskService:    diskService,
		stateService:   stateService,
		dlqService:     dlqService,
		metricsService: metricsService,
		encoder:        encoder,
		validator:      validator,
		cache:          cache,
		maxRetries:     maxRetries,
	}
}

func (uc *TranscodeUseCase) Execute(ctx context.Context, job *types.Job) error {
	var lastError error
	var errorCode string

	for attempt := 0; attempt <= uc.maxRetries; attempt++ {
		if attempt > 0 {
			// Get retry strategy based on error code
			strategy := types.GetRetryStrategy(errorCode)
			
			if !strategy.ShouldRetry {
				log.Printf("Job %s: Error type not retryable (%s)", job.ID, errorCode)
				uc.handleError(job.ID, "validation", errorCode, fmt.Sprintf("Non-retryable error: %v", lastError))
				return lastError
			}

			if attempt > len(strategy.Backoff) {
				log.Printf("Job %s: Max retries reached for error type %s", job.ID, errorCode)
				uc.handleError(job.ID, "retry", errorCode, fmt.Sprintf("Max retries exceeded: %v", lastError))
				return lastError
			}

			delay := strategy.Backoff[attempt-1]
			log.Printf("Job %s: Retrying after %v (attempt %d, error: %s)", job.ID, delay, attempt+1, errorCode)
			time.Sleep(delay)
			uc.jobService.IncrementRetry(job.ID)
		}

		log.Printf("Job %s: Processing (attempt %d/%d)", job.ID, attempt+1, uc.maxRetries+1)

		err := uc.process(ctx, job)
		if err == nil {
			return nil
		}

		lastError = err
		errorCode = extractErrorCode(err)
		log.Printf("Job %s: Attempt %d failed with code %s: %v", job.ID, attempt+1, errorCode, err)
	}

	// Final failure - move to DLQ
	uc.handleError(job.ID, "retry", errorCode, fmt.Sprintf("Max retries exceeded: %v", lastError))
	
	// Move to DLQ
	if jobData, err := uc.jobService.GetJob(job.ID); err == nil {
		uc.dlqService.MoveToDLQ(jobData, fmt.Sprintf("Max retries exceeded (%d attempts)", uc.maxRetries+1))
	}
	
	return lastError
}

func extractErrorCode(err error) string {
	// Try to extract error code from error message
	// This is a simple implementation - you might want to use typed errors
	errStr := err.Error()
	
	if contains(errStr, "resolution") {
		return domain.CodeResolutionDetectFailed
	}
	if contains(errStr, "disk") {
		return domain.CodeDiskFull
	}
	if contains(errStr, "permission") {
		return domain.CodePermissionDenied
	}
	if contains(errStr, "I/O") || contains(errStr, "io") {
		return domain.CodeIOError
	}
	
	return domain.CodeFFmpegFailed
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && 
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || 
		len(s) > len(substr)+1 && s[1:len(substr)+1] == substr))
}

// Step functions for idempotent encoding

func (uc *TranscodeUseCase) stepHashComputation(job *types.Job, state *types.JobState) error {
	log.Printf("Job %s: Step - Hash computation", job.ID)
	uc.jobService.UpdateProgress(job.ID, 5, "Computing file hash")

	hash, err := uc.cache.ComputeHash(job.InputPath)
	if err != nil {
		return fmt.Errorf("failed to compute hash: %w", err)
	}

	if err := uc.jobService.SetHash(job.ID, hash); err != nil {
		return err
	}

	state.MarkStepCompleted(string(types.StepHashComputation))
	state.CurrentStep = string(types.StepDeduplication)
	return uc.stateService.SaveState(state)
}

func (uc *TranscodeUseCase) stepDeduplication(job *types.Job, state *types.JobState) (bool, error) {
	log.Printf("Job %s: Step - Deduplication check", job.ID)
	uc.jobService.UpdateProgress(job.ID, 10, "Checking for duplicates")

	// Get hash from job
	jobData, _ := uc.jobService.GetJob(job.ID)
	if jobData.InputHash == "" {
		return false, fmt.Errorf("hash not computed")
	}

	if existingJobID, found := uc.cache.CheckDuplicate(jobData.InputHash); found {
		log.Printf("Job %s: Duplicate detected, reusing output from job %s", job.ID, existingJobID)
		uc.jobService.UpdateProgress(job.ID, 100, "Completed (duplicate)")
		uc.jobService.UpdateStatus(job.ID, types.StatusCompleted)
		
		state.Status = "completed"
		state.MarkStepCompleted(string(types.StepDeduplication))
		uc.stateService.SaveState(state)
		return true, nil
	}

	state.MarkStepCompleted(string(types.StepDeduplication))
	state.CurrentStep = string(types.StepAnalysis)
	return false, uc.stateService.SaveState(state)
}

func (uc *TranscodeUseCase) stepAnalysis(job *types.Job, state *types.JobState) (*types.VideoInfo, error) {
	log.Printf("Job %s: Step - Video analysis", job.ID)
	uc.jobService.UpdateProgress(job.ID, 15, "Analyzing input video")

	videoInfo, err := uc.encoder.GetVideoInfo(job.InputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze video: %w", err)
	}

	state.MarkStepCompleted(string(types.StepAnalysis))
	state.CurrentStep = string(types.StepEncoding720p) // Start with first quality
	uc.stateService.SaveState(state)

	return videoInfo, nil
}

func (uc *TranscodeUseCase) stepEncoding(job *types.Job, state *types.JobState, videoInfo *types.VideoInfo) (*types.EncodeResult, []types.QualityVariant, error) {
	log.Printf("Job %s: Step - Multi-quality encoding", job.ID)
	uc.jobService.UpdateProgress(job.ID, 30, "Starting multi-quality encoding")

	// Check disk limit
	if err := uc.checkJobDiskLimit(job.ID); err != nil {
		return nil, nil, err
	}

	// Encode with resume support (skips completed qualities)
	result, variants, err := uc.encodeWithResume(job, state, videoInfo)
	
	// Check if it's a partial success
	isPartialSuccess := false
	var failedQualities []string
	
	if err != nil {
		if partialErr, ok := err.(*ffmpeg.PartialSuccessError); ok {
			isPartialSuccess = true
			failedQualities = partialErr.FailedQualities
			log.Printf("Job %s: Partial success - %d/%d qualities succeeded", 
				job.ID, partialErr.SuccessCount, partialErr.TotalCount)
		} else {
			return nil, nil, fmt.Errorf("encoding failed: %w", err)
		}
	}

	// Save partial success info to state
	if isPartialSuccess {
		for _, quality := range failedQualities {
			state.MarkQualityFailed(quality)
		}
		uc.stateService.SaveState(state)
	}

	state.MarkStepCompleted(string(types.StepMasterPlaylist))
	state.CurrentStep = string(types.StepCleanup)
	uc.stateService.SaveState(state)

	return result, variants, err
}

func (uc *TranscodeUseCase) stepFinalization(job *types.Job, state *types.JobState, result *types.EncodeResult, variants []types.QualityVariant, videoInfo *types.VideoInfo) error {
	log.Printf("Job %s: Step - Finalization", job.ID)
	uc.jobService.UpdateProgress(job.ID, 90, "Finalizing outputs")

	inputMeta := buildInputMetadata(job.InputPath, videoInfo, variants)
	outputInfo := buildOutputInfo(job.ID, result)

	if err := uc.jobService.SetMetadata(job.ID, inputMeta, outputInfo); err != nil {
		return err
	}

	// Set error info if partial success
	isPartialSuccess := len(state.QualitiesFailed) > 0
	if isPartialSuccess {
		jobErr := &types.JobError{
			Code:              "ERR_PARTIAL_ENCODING",
			Stage:             "encoding",
			Message:           fmt.Sprintf("Some qualities failed to encode: %v", state.QualitiesFailed),
			FailedResolutions: state.QualitiesFailed,
			LogsPath:          fmt.Sprintf("outputs/%s/logs", job.ID),
		}
		uc.jobService.SetError(job.ID, jobErr)
	}

	// Cleanup intermediate files
	if !state.IsStepCompleted(string(types.StepCleanup)) {
		if err := uc.cleanupIntermediateFiles(job.ID); err != nil {
			log.Printf("Warning: Failed to cleanup intermediate files for job %s: %v", job.ID, err)
		}
		state.MarkStepCompleted(string(types.StepCleanup))
		uc.stateService.SaveState(state)
	}

	// Set final status
	if isPartialSuccess {
		uc.jobService.UpdateProgress(job.ID, 100, "Completed with partial success")
		uc.jobService.UpdateStatus(job.ID, types.StatusPartialSuccess)
		state.Status = "partial_success"
		log.Printf("Job %s completed with partial success", job.ID)
	} else {
		uc.jobService.UpdateProgress(job.ID, 100, "Completed")
		uc.jobService.UpdateStatus(job.ID, types.StatusCompleted)
		state.Status = "completed"
		log.Printf("Job %s completed successfully", job.ID)
	}

	// Store in cache
	jobData, _ := uc.jobService.GetJob(job.ID)
	if jobData.InputHash != "" {
		uc.cache.Store(jobData.InputHash, job.ID)
	}

	state.MarkStepCompleted(string(types.StepFinalization))
	uc.stateService.SaveState(state)

	// End metrics tracking
	qualitiesCount := len(state.QualitiesComplete)
	chunksCount := 0
	if outputInfo != nil {
		for _, q := range outputInfo.Qualities {
			chunksCount += q.ChunkCount
		}
	}
	
	finalStatus := "completed"
	if isPartialSuccess {
		finalStatus = "partial_success"
	}
	
	uc.metricsService.EndJobMetrics(job.ID, finalStatus, qualitiesCount, chunksCount)

	return nil
}

func (uc *TranscodeUseCase) encodeWithResume(job *types.Job, state *types.JobState, videoInfo *types.VideoInfo) (*types.EncodeResult, []types.QualityVariant, error) {
	// Select variants
	variants := uc.encoder.SelectVariants(videoInfo.Width, videoInfo.Height)
	
	result := &types.EncodeResult{
		QualityOutputs: []types.QualityOutputInfo{},
	}

	var successfulVariants []types.QualityVariant
	var failedQualities []string

	// Encode each quality with idempotency
	for _, variant := range variants {
		// Skip if already completed
		if state.IsQualityCompleted(variant.Name) {
			log.Printf("Job %s: Skipping %s (already completed)", job.ID, variant.Name)
			
			// Load existing output info
			qualityResult := uc.loadExistingQuality(job.ID, variant)
			if qualityResult != nil {
				result.QualityOutputs = append(result.QualityOutputs, *qualityResult)
				successfulVariants = append(successfulVariants, variant)
			}
			continue
		}

		log.Printf("Job %s: Encoding %s...", job.ID, variant.Name)
		state.CurrentStep = fmt.Sprintf("encoding_%s", variant.Name)
		uc.stateService.SaveState(state)

		qualityResult := ffmpeg.EncodeQuality(job.ID, job.InputPath, variant, job.ChunkDuration, uc.diskService.GetOutputsDir(), videoInfo.HasAudio)
		
		if qualityResult.Success {
			result.QualityOutputs = append(result.QualityOutputs, types.QualityOutputInfo{
				Quality:    qualityResult.Quality,
				Playlist:   qualityResult.Playlist,
				ChunksDir:  qualityResult.ChunksDir,
				ChunkCount: qualityResult.ChunkCount,
			})
			successfulVariants = append(successfulVariants, variant)
			
			// Mark as completed
			state.MarkQualityCompleted(variant.Name)
			uc.stateService.SaveState(state)
		} else {
			log.Printf("Failed to encode %s: %v", variant.Name, qualityResult.Error)
			failedQualities = append(failedQualities, variant.Name)
			state.MarkQualityFailed(variant.Name)
			uc.stateService.SaveState(state)
		}
	}

	// Generate master playlist
	if len(successfulVariants) > 0 {
		hlsDir := filepath.Join(uc.diskService.GetOutputsDir(), job.ID, "hls")
		masterPlaylist := filepath.Join(hlsDir, "master.m3u8")
		
		if err := ffmpeg.GenerateMasterPlaylist(masterPlaylist, successfulVariants, videoInfo.HasAudio); err != nil {
			return nil, nil, fmt.Errorf("failed to generate master playlist: %w", err)
		}
		
		result.MasterPlaylist = masterPlaylist
	}

	// Return partial success if some failed
	if len(failedQualities) > 0 && len(successfulVariants) > 0 {
		return result, successfulVariants, &ffmpeg.PartialSuccessError{
			FailedQualities: failedQualities,
			SuccessCount:    len(successfulVariants),
			TotalCount:      len(variants),
		}
	}

	if len(successfulVariants) == 0 {
		return nil, nil, fmt.Errorf("all quality encodings failed")
	}

	return result, successfulVariants, nil
}

func (uc *TranscodeUseCase) loadExistingQuality(jobID string, variant types.QualityVariant) *types.QualityOutputInfo {
	outputsDir := uc.diskService.GetOutputsDir()
	variantDir := filepath.Join(outputsDir, jobID, "hls", variant.Name)
	
	if !utils.FileExists(variantDir) {
		return nil
	}

	chunks, _ := filepath.Glob(filepath.Join(variantDir, "chunk_*.ts"))
	
	return &types.QualityOutputInfo{
		Quality:    variant.Name,
		Playlist:   filepath.Join(variantDir, "main.m3u8"),
		ChunksDir:  variantDir,
		ChunkCount: len(chunks),
	}
}

func (uc *TranscodeUseCase) process(ctx context.Context, job *types.Job) error {
	// Start metrics tracking
	uc.metricsService.StartJobMetrics(job.ID, job.InputPath)
	
	// Start metrics monitoring
	stopMetrics := make(chan bool)
	go uc.metricsService.StartMetricsMonitoring(job.ID, stopMetrics)
	defer func() { stopMetrics <- true }()

	// Create or load state
	state, err := uc.getOrCreateState(job)
	if err != nil {
		return fmt.Errorf("failed to get/create state: %w", err)
	}

	// Start heartbeat
	stopHeartbeat := make(chan bool)
	go uc.startHeartbeat(job.ID, stopHeartbeat)
	defer func() { stopHeartbeat <- true }()

	// Update status to running
	if err := uc.jobService.UpdateStatus(job.ID, types.StatusRunning); err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	// Execute steps with resume support
	return uc.executeSteps(ctx, job, state)
}

func (uc *TranscodeUseCase) getOrCreateState(job *types.Job) (*types.JobState, error) {
	// Try to load existing state
	state, err := uc.stateService.LoadState(job.ID)
	if err == nil {
		log.Printf("Job %s: Resuming from step %s", job.ID, state.CurrentStep)
		return state, nil
	}

	// Create new state
	return uc.stateService.CreateState(job.ID, job.InputPath, job.ChunkDuration)
}

func (uc *TranscodeUseCase) startHeartbeat(jobID string, stop chan bool) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := uc.stateService.UpdateHeartbeat(jobID); err != nil {
				log.Printf("Warning: Failed to update heartbeat for job %s: %v", jobID, err)
			}
		case <-stop:
			return
		}
	}
}

func (uc *TranscodeUseCase) executeSteps(ctx context.Context, job *types.Job, state *types.JobState) error {
	// Step 1: Hash computation (idempotent)
	if !state.IsStepCompleted(string(types.StepHashComputation)) {
		if err := uc.stepHashComputation(job, state); err != nil {
			return err
		}
	}

	// Step 2: Deduplication check (idempotent)
	if !state.IsStepCompleted(string(types.StepDeduplication)) {
		isDuplicate, err := uc.stepDeduplication(job, state)
		if err != nil {
			return err
		}
		if isDuplicate {
			return nil // Job completed via deduplication
		}
	}

	// Step 3: Analysis (idempotent)
	var videoInfo *types.VideoInfo
	if !state.IsStepCompleted(string(types.StepAnalysis)) {
		var err error
		videoInfo, err = uc.stepAnalysis(job, state)
		if err != nil {
			return err
		}
	} else {
		// Reload video info if resuming
		var err error
		videoInfo, err = uc.encoder.GetVideoInfo(job.InputPath)
		if err != nil {
			return fmt.Errorf("failed to reload video info: %w", err)
		}
	}

	// Step 4: Encoding (idempotent per quality)
	result, variants, err := uc.stepEncoding(job, state, videoInfo)
	if err != nil {
		return err
	}

	// Step 5: Finalization
	return uc.stepFinalization(job, state, result, variants, videoInfo)
}

func (uc *TranscodeUseCase) handleError(jobID, stage, code, message string) {
	jobErr := &types.JobError{
		Code:     code,
		Stage:    stage,
		Message:  message,
		LogsPath: fmt.Sprintf("logs/%s/%s.log", jobID, stage),
	}

	uc.jobService.SetError(jobID, jobErr)
}

func buildInputMetadata(inputPath string, videoInfo *types.VideoInfo, variants []types.QualityVariant) *types.VideoMetadata {
	generatedQualities := make([]types.QualityMetadata, len(variants))
	for i, v := range variants {
		generatedQualities[i] = types.QualityMetadata{
			Quality:    v.Name,
			Resolution: fmt.Sprintf("%dx%d", v.Width, v.Height),
			Bitrate:    v.Bitrate,
		}
	}

	return &types.VideoMetadata{
		FileName:           filepath.Base(inputPath),
		SourcePath:         inputPath,
		Resolution:         fmt.Sprintf("%dx%d", videoInfo.Width, videoInfo.Height),
		Duration:           videoInfo.Duration,
		GeneratedQualities: generatedQualities,
	}
}

func buildOutputInfo(jobID string, result *types.EncodeResult) *types.OutputInfo {
	qualities := make([]types.QualityOutput, len(result.QualityOutputs))
	for i, qo := range result.QualityOutputs {
		qualities[i] = types.QualityOutput{
			Quality:    qo.Quality,
			Playlist:   qo.Playlist,
			ChunksDir:  qo.ChunksDir,
			ChunkCount: qo.ChunkCount,
		}
	}

	return &types.OutputInfo{
		OutputDir:      filepath.Dir(result.MasterPlaylist),
		MasterPlaylist: result.MasterPlaylist,
		Qualities:      qualities,
	}
}

func (uc *TranscodeUseCase) checkJobDiskLimit(jobID string) error {
	jobSize, err := uc.diskService.GetJobSize(jobID)
	if err != nil {
		return fmt.Errorf("failed to get job size: %w", err)
	}

	maxSizeMB := float64(1500) // MaxJobSizeMB from constants
	jobSizeMB := float64(jobSize) / (1024 * 1024)

	if jobSizeMB > maxSizeMB {
		return fmt.Errorf("job size (%.2f MB) exceeds limit (%.2f MB)", jobSizeMB, maxSizeMB)
	}

	return nil
}

func (uc *TranscodeUseCase) cleanupIntermediateFiles(jobID string) error {
	// This will be implemented in disk service
	return uc.diskService.CleanupIntermediateFiles(jobID)
}
