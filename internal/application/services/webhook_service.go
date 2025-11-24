package services

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoder-service/pkg/config"
	"encoder-service/pkg/types"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// WebhookService handles HTTP callback notifications
type WebhookService struct {
	config     *config.Config
	httpClient *http.Client
}

// WebhookPayload represents the data sent to webhook URLs
type WebhookPayload struct {
	JobID       string                 `json:"jobId"`
	Status      string                 `json:"status"`
	InputPath   string                 `json:"inputPath"`
	OutputURL   string                 `json:"outputUrl,omitempty"`
	CompletedAt *time.Time             `json:"completedAt,omitempty"`
	Error       *types.JobError        `json:"error,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
}

// WebhookConfig represents webhook configuration for a job
type WebhookConfig struct {
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
	Secret  string            `json:"secret,omitempty"`
	Retries int               `json:"retries,omitempty"`
}

func NewWebhookService(cfg *config.Config) *WebhookService {
	return &WebhookService{
		config: cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SendWebhook sends a webhook notification with retry logic
func (s *WebhookService) SendWebhook(webhookConfig WebhookConfig, payload WebhookPayload) error {
	payload.Timestamp = time.Now()

	maxRetries := webhookConfig.Retries
	if maxRetries == 0 {
		maxRetries = 3
	}

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			delay := time.Duration(attempt*attempt) * time.Second
			log.Printf("Webhook retry %d/%d after %v for job %s", attempt, maxRetries, delay, payload.JobID)
			time.Sleep(delay)
		}

		err := s.sendWebhookRequest(webhookConfig, payload)
		if err == nil {
			log.Printf("Webhook sent successfully for job %s to %s", payload.JobID, webhookConfig.URL)
			return nil
		}

		lastErr = err
		log.Printf("Webhook attempt %d failed for job %s: %v", attempt+1, payload.JobID, err)
	}

	return fmt.Errorf("webhook failed after %d attempts: %w", maxRetries+1, lastErr)
}

// SendJobCompletedWebhook sends webhook for completed job
func (s *WebhookService) SendJobCompletedWebhook(job *types.Job, webhookConfig WebhookConfig, outputURL string) error {
	payload := WebhookPayload{
		JobID:       job.ID,
		Status:      string(job.Status),
		InputPath:   job.InputPath,
		OutputURL:   outputURL,
		CompletedAt: job.Meta.CompletedAt,
		Error:       job.Error,
		Metadata: map[string]interface{}{
			"processingTimeSec": job.Meta.ProcessingTimeSec,
			"retries":           job.Retries,
			"chunkDuration":     job.ChunkDuration,
		},
	}

	return s.SendWebhook(webhookConfig, payload)
}

// SendJobFailedWebhook sends webhook for failed job
func (s *WebhookService) SendJobFailedWebhook(job *types.Job, webhookConfig WebhookConfig) error {
	payload := WebhookPayload{
		JobID:     job.ID,
		Status:    string(job.Status),
		InputPath: job.InputPath,
		Error:     job.Error,
		Metadata: map[string]interface{}{
			"retries":    job.Retries,
			"maxRetries": job.MaxRetries,
		},
	}

	return s.SendWebhook(webhookConfig, payload)
}

func (s *WebhookService) sendWebhookRequest(webhookConfig WebhookConfig, payload WebhookPayload) error {
	// Marshal payload
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	// Create request
	req, err := http.NewRequest("POST", webhookConfig.URL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create webhook request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "VideoEncoderService/1.0")

	// Add custom headers
	for key, value := range webhookConfig.Headers {
		req.Header.Set(key, value)
	}

	// Add signature if secret is provided
	if webhookConfig.Secret != "" {
		signature := s.generateSignature(jsonData, webhookConfig.Secret)
		req.Header.Set("X-Webhook-Signature", signature)
	}

	// Send request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned non-success status: %d", resp.StatusCode)
	}

	return nil
}

func (s *WebhookService) generateSignature(data []byte, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(data)
	return "sha256=" + hex.EncodeToString(h.Sum(nil))
}
