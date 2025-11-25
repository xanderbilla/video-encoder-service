package main

import (
	"context"
	"encoder-service/internal/api/handlers"
	"encoder-service/internal/api/middleware"
	"encoder-service/internal/api/routes"
	"encoder-service/internal/application/services"
	"encoder-service/internal/application/usecases"
	"encoder-service/internal/infrastructure/cache"
	"encoder-service/internal/infrastructure/ffmpeg"
	"encoder-service/internal/infrastructure/filesystem"
	"encoder-service/internal/infrastructure/websocket"
	"encoder-service/pkg/config"
	"encoder-service/pkg/types"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	// Load configuration
	cfg := config.Load()

	log.Printf("Starting Video Encoder Service v2.0.0")
	log.Printf("Configuration loaded: Port=%s, MaxWorkers=%d, MaxRetries=%d", 
		cfg.Server.Port, cfg.Worker.MaxWorkers, cfg.Worker.MaxRetries)

	// Initialize infrastructure
	jobRepo := filesystem.NewInMemoryJobRepository()
	validator := filesystem.NewFileValidator()
	encoder := ffmpeg.NewHLSEncoder(cfg.Storage.OutputsDir)
	dedupCache := cache.NewDeduplicationCache(cfg.Storage.CacheDir)

	// Initialize services
	jobService := services.NewJobService(jobRepo, cfg)
	diskService := services.NewDiskService(cfg)
	stateService := services.NewStateService(cfg)
	dlqService := services.NewDLQService(cfg)
	metricsService := services.NewMetricsService(cfg)
	webhookService := services.NewWebhookService(cfg)
	batchService := services.NewBatchService(cfg)
	cdnService := services.NewCDNService(cfg)

	// Create job queue
	jobQueue := make(chan *types.Job, 100)

	// Initialize scheduler service
	schedulerService := services.NewSchedulerService(cfg, jobQueue)

	// Initialize WebSocket manager
	wsManager := websocket.NewManager()

	// Initialize use cases
	transcodeUseCase := usecases.NewTranscodeUseCase(
		jobService,
		diskService,
		stateService,
		dlqService,
		metricsService,
		wsManager,
		encoder,
		validator,
		dedupCache,
		cfg.Worker.MaxRetries,
	)

	// Initialize handlers
	jobHandler := handlers.NewJobHandler(
		jobService,
		diskService,
		validator,
		transcodeUseCase,
		webhookService,
		schedulerService,
		batchService,
		cdnService,
		wsManager,
		jobQueue,
		cfg.Worker.MaxWorkers,
	)

	// Start background workers
	diskService.StartCleanupWorker()
	log.Println("Disk cleanup worker started")

	// Resume orphaned jobs
	go resumeOrphanedJobs(stateService, jobQueue, jobService)

	// Setup graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Start scheduler service
	schedulerService.Start(ctx)
	log.Println("Scheduler service started")

	// Start job processing workers
	for i := 0; i < cfg.Worker.MaxWorkers; i++ {
		go jobHandler.StartWorker(ctx)
		log.Printf("Worker %d started", i+1)
	}

	// Initialize rate limiter (100 requests per minute, burst of 20)
	rateLimiter := middleware.NewRateLimiter(100, 20)

	// Setup router
	router := routes.SetupRoutes(jobHandler, rateLimiter)

	// Start server
	server := &http.Server{
		Addr:    ":" + cfg.Server.Port,
		Handler: router,
	}

	go func() {
		log.Printf("API server listening on port %s", cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	// Wait for interrupt signal
	<-ctx.Done()
	log.Println("Shutdown signal received")

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.Server.ShutdownTimeout)*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	log.Println("Server stopped gracefully")
}

func resumeOrphanedJobs(stateService *services.StateService, jobQueue chan *types.Job, jobService *services.JobService) {
	log.Println("Checking for orphaned jobs...")
	
	orphaned, err := stateService.FindOrphanedJobs()
	if err != nil {
		log.Printf("Error finding orphaned jobs: %v", err)
		return
	}

	if len(orphaned) == 0 {
		log.Println("No orphaned jobs found")
		return
	}

	log.Printf("Found %d orphaned jobs, resuming...", len(orphaned))
	for _, state := range orphaned {
		log.Printf("Resuming job %s from step %s", state.JobID, state.CurrentStep)
		
		// Get the job and requeue it
		if job, err := jobService.GetJob(state.JobID); err == nil {
			jobService.UpdateStatus(state.JobID, types.StatusQueued)
			jobQueue <- job
		}
	}
}
