package types

import "time"

// JobState represents the persistent state of a job
type JobState struct {
	JobID             string    `json:"jobId"`
	Status            string    `json:"status"`
	CurrentStep       string    `json:"currentStep"`
	StepsCompleted    []string  `json:"stepsCompleted"`
	QualitiesComplete []string  `json:"qualitiesComplete"`
	QualitiesFailed   []string  `json:"qualitiesFailed"`
	ChunksCompleted   int       `json:"chunksCompleted"`
	TotalChunks       int       `json:"totalChunks"`
	LastHeartbeat     time.Time `json:"lastHeartbeat"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
	ResumeSupported   bool      `json:"resumeSupported"`
	InputPath         string    `json:"inputPath"`
	ChunkDuration     int       `json:"chunkDuration"`
}

// EncodingStep represents a step in the encoding process
type EncodingStep string

const (
	StepValidation       EncodingStep = "validation"
	StepAnalysis         EncodingStep = "analysis"
	StepHashComputation  EncodingStep = "hash_computation"
	StepDeduplication    EncodingStep = "deduplication"
	StepEncoding240p     EncodingStep = "encoding_240p"
	StepEncoding360p     EncodingStep = "encoding_360p"
	StepEncoding480p     EncodingStep = "encoding_480p"
	StepEncoding720p     EncodingStep = "encoding_720p"
	StepEncoding1080p    EncodingStep = "encoding_1080p"
	StepEncoding1440p    EncodingStep = "encoding_1440p"
	StepEncoding4K       EncodingStep = "encoding_4k"
	StepMasterPlaylist   EncodingStep = "master_playlist"
	StepCleanup          EncodingStep = "cleanup"
	StepFinalization     EncodingStep = "finalization"
)

// IsStepCompleted checks if a step is completed
func (s *JobState) IsStepCompleted(step string) bool {
	for _, completed := range s.StepsCompleted {
		if completed == step {
			return true
		}
	}
	return false
}

// IsQualityCompleted checks if a quality is completed
func (s *JobState) IsQualityCompleted(quality string) bool {
	for _, completed := range s.QualitiesComplete {
		if completed == quality {
			return true
		}
	}
	return false
}

// MarkStepCompleted marks a step as completed
func (s *JobState) MarkStepCompleted(step string) {
	if !s.IsStepCompleted(step) {
		s.StepsCompleted = append(s.StepsCompleted, step)
	}
	s.UpdatedAt = time.Now()
}

// MarkQualityCompleted marks a quality as completed
func (s *JobState) MarkQualityCompleted(quality string) {
	if !s.IsQualityCompleted(quality) {
		s.QualitiesComplete = append(s.QualitiesComplete, quality)
	}
	s.UpdatedAt = time.Now()
}

// MarkQualityFailed marks a quality as failed
func (s *JobState) MarkQualityFailed(quality string) {
	s.QualitiesFailed = append(s.QualitiesFailed, quality)
	s.UpdatedAt = time.Now()
}

// UpdateHeartbeat updates the last heartbeat timestamp
func (s *JobState) UpdateHeartbeat() {
	s.LastHeartbeat = time.Now()
	s.UpdatedAt = time.Now()
}

// IsOrphaned checks if the job is orphaned (no heartbeat for 60s)
func (s *JobState) IsOrphaned() bool {
	return time.Since(s.LastHeartbeat) > 60*time.Second
}
