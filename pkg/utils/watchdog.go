package utils

import (
	"context"
	"fmt"
	"time"
)

// Watchdog monitors a task and cancels it if it exceeds the timeout
type Watchdog struct {
	timeout time.Duration
}

// NewWatchdog creates a new watchdog with calculated timeout
// timeout = videoDuration * multiplier + buffer
func NewWatchdog(videoDuration float64) *Watchdog {
	// Calculate timeout: 2x video duration + 60s buffer
	timeout := time.Duration(videoDuration*2)*time.Second + 60*time.Second
	
	// Minimum timeout of 2 minutes
	if timeout < 2*time.Minute {
		timeout = 2 * time.Minute
	}

	return &Watchdog{
		timeout: timeout,
	}
}

// NewWatchdogWithTimeout creates a watchdog with explicit timeout
func NewWatchdogWithTimeout(timeout time.Duration) *Watchdog {
	return &Watchdog{
		timeout: timeout,
	}
}

// Watch executes a function with timeout monitoring
func (w *Watchdog) Watch(ctx context.Context, fn func() error) error {
	// Create context with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, w.timeout)
	defer cancel()

	// Channel to receive result
	done := make(chan error, 1)

	// Execute function in goroutine
	go func() {
		done <- fn()
	}()

	// Wait for completion or timeout
	select {
	case err := <-done:
		return err
	case <-timeoutCtx.Done():
		return fmt.Errorf("operation timed out after %v", w.timeout)
	}
}

// GetTimeout returns the configured timeout
func (w *Watchdog) GetTimeout() time.Duration {
	return w.timeout
}
