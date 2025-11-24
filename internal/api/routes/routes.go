package routes

import (
	"encoder-service/internal/api/handlers"
	"encoder-service/internal/api/middleware"

	"github.com/gorilla/mux"
)

func SetupRoutes(jobHandler *handlers.JobHandler, rateLimiter *middleware.RateLimiter) *mux.Router {
	r := mux.NewRouter()

	// Global Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.Logging)
	
	// Rate limiting middleware
	if rateLimiter != nil {
		r.Use(rateLimiter.Middleware)
	}

	// Health endpoint (no rate limit)
	r.HandleFunc("/health", jobHandler.Health).Methods("GET")

	// Job endpoints
	r.HandleFunc("/transcode", jobHandler.CreateJob).Methods("POST")
	r.HandleFunc("/jobs", jobHandler.ListJobs).Methods("GET")
	r.HandleFunc("/jobs/{id}", jobHandler.GetJob).Methods("GET")
	r.HandleFunc("/jobs/{id}", jobHandler.DeleteJob).Methods("DELETE")
	r.HandleFunc("/jobs/{id}/resume", jobHandler.ResumeJob).Methods("POST")
	
	// WebSocket endpoint for real-time progress
	r.HandleFunc("/jobs/{id}/progress", jobHandler.HandleWebSocket).Methods("GET")
	
	// Batch processing endpoints
	r.HandleFunc("/batch", jobHandler.CreateBatchJob).Methods("POST")
	r.HandleFunc("/batch/{id}", jobHandler.GetBatchStatus).Methods("GET")
	
	// Scheduling endpoint
	r.HandleFunc("/schedule", jobHandler.ScheduleJob).Methods("POST")
	
	// Video analysis endpoint
	r.HandleFunc("/analyze", jobHandler.GetDetailedVideoInfo).Methods("POST")

	return r
}
