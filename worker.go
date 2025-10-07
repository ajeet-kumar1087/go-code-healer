package healer

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ajeet-kumar1087/go-code-healer/ai"
)

// BackgroundWorker handles background processing of panic events
type BackgroundWorker struct {
	id        int
	healer    *Healer
	logger    Logger
	stopCh    chan struct{}
	wg        *sync.WaitGroup
	isRunning bool
	mu        sync.RWMutex
}

// NewBackgroundWorker creates a new background worker
func NewBackgroundWorker(id int, healer *Healer, logger Logger, wg *sync.WaitGroup) *BackgroundWorker {
	return &BackgroundWorker{
		id:     id,
		healer: healer,
		logger: logger,
		stopCh: make(chan struct{}),
		wg:     wg,
	}
}

// Start begins processing panic events from the queue
func (w *BackgroundWorker) Start(ctx context.Context) error {
	w.mu.Lock()
	if w.isRunning {
		w.mu.Unlock()
		return nil // Already running
	}
	w.isRunning = true
	w.mu.Unlock()

	w.wg.Add(1)
	go w.run(ctx)

	if w.logger != nil {
		w.logger.Debug("Background worker %d started", w.id)
	}

	return nil
}

// Stop gracefully stops the worker
func (w *BackgroundWorker) Stop() error {
	w.mu.Lock()
	if !w.isRunning {
		w.mu.Unlock()
		return nil // Already stopped
	}
	w.isRunning = false
	w.mu.Unlock()

	close(w.stopCh)

	if w.logger != nil {
		w.logger.Debug("Background worker %d stopping", w.id)
	}

	return nil
}

// run is the main worker loop
func (w *BackgroundWorker) run(ctx context.Context) {
	defer w.wg.Done()

	if w.logger != nil {
		w.logger.Debug("Worker %d started processing", w.id)
	}

	for {
		select {
		case <-ctx.Done():
			if w.logger != nil {
				w.logger.Debug("Worker %d stopped due to context cancellation", w.id)
			}
			return

		case <-w.stopCh:
			if w.logger != nil {
				w.logger.Debug("Worker %d stopped due to stop signal", w.id)
			}
			return

		case event := <-w.healer.errorQueue:
			w.processEvent(ctx, event)
		}
	}
}

// processEvent processes a single panic event
func (w *BackgroundWorker) processEvent(ctx context.Context, event PanicEvent) {
	if w.logger != nil {
		w.logger.Debug("Worker %d processing event %s", w.id, event.ID)
	}

	// Update event status
	event.Status = "processing"
	now := time.Now()
	event.ProcessedAt = &now

	// Process the event with retry logic and circuit breaker
	err := w.processEventWithRetry(ctx, event)
	if err != nil {
		event.Status = "failed"
		if w.logger != nil {
			w.logger.Error("Worker %d failed to process event %s: %v", w.id, event.ID, err)
		}
	} else {
		event.Status = "completed"
		if w.logger != nil {
			w.logger.Info("Worker %d successfully processed event %s", w.id, event.ID)
		}
	}
}

// processEventWithRetry processes an event with retry logic and circuit breaker
func (w *BackgroundWorker) processEventWithRetry(ctx context.Context, event PanicEvent) error {
	// Use retry manager for processing
	return w.healer.retryManager.ExecuteWithRetry(ctx, fmt.Sprintf("process-event-%s", event.ID), func() error {
		// Use circuit breaker for external API calls
		return w.healer.circuitBreaker.Execute(ctx, "event-processing", func() error {
			// Use enhanced timeout management for different processing phases
			return w.processEventWithTimeoutManagement(ctx, event)
		})
	})
}

// processEventWithAI processes an event using AI fix generation
func (w *BackgroundWorker) processEventWithAI(ctx context.Context, event PanicEvent) (*FixResponse, error) {
	// Create timeout context for AI processing
	aiCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	if w.logger != nil {
		w.logger.Debug("Worker %d starting AI processing for event %s", w.id, event.ID)
	}

	// Check if provider manager is available
	if w.healer.providerManager == nil {
		if w.logger != nil {
			w.logger.Debug("Provider manager not available, skipping AI processing for event %s", event.ID)
		}
		return nil, nil // Not an error, just skip AI processing
	}

	// Create fix request from panic event
	fixRequest := ai.FixRequest{
		Error:      event.Error,
		StackTrace: event.StackTrace,
		SourceCode: w.extractSourceCode(event),
		Context:    event.GetContext(),
	}

	// Generate fix using provider manager with timeout management
	fixResponse, err := w.healer.providerManager.GenerateFixWithFallback(aiCtx, fixRequest)
	if err != nil {
		// Check if it's a timeout or cancellation
		if ctx.Err() != nil {
			return nil, fmt.Errorf("AI processing cancelled: %w", ctx.Err())
		}
		return nil, fmt.Errorf("AI fix generation failed: %w", err)
	}

	if w.logger != nil {
		w.logger.Info("Worker %d generated AI fix for event %s (confidence: %.2f, valid: %v)",
			w.id, event.ID, fixResponse.Confidence, fixResponse.IsValid)
	}

	// Store the fix response for logging
	w.storeFixResponse(event, fixResponse)

	return fixResponse, nil
}

// processEventWithGit processes an event using Git operations to create pull requests
func (w *BackgroundWorker) processEventWithGit(ctx context.Context, event PanicEvent, fixResponse *FixResponse) error {
	// Create timeout context for Git processing
	gitCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()

	if w.logger != nil {
		w.logger.Debug("Worker %d starting Git processing for event %s", w.id, event.ID)
	}

	// Check if Git client is available
	if w.healer.gitClient == nil {
		if w.logger != nil {
			w.logger.Debug("Git client not available, skipping Git processing for event %s", event.ID)
		}
		return nil // Not an error, just skip Git processing
	}

	// Skip Git processing if we don't have a valid AI fix
	if fixResponse == nil || !fixResponse.IsValid || fixResponse.ProposedFix == "" {
		if w.logger != nil {
			w.logger.Debug("No valid AI fix available, skipping Git processing for event %s", event.ID)
		}
		return nil
	}

	// Check confidence threshold (only create PRs for high-confidence fixes)
	confidenceThreshold := 0.7 // 70% confidence threshold
	if fixResponse.Confidence < confidenceThreshold {
		if w.logger != nil {
			w.logger.Debug("AI fix confidence (%.2f) below threshold (%.2f), skipping Git processing for event %s",
				fixResponse.Confidence, confidenceThreshold, event.ID)
		}
		return nil
	}

	// Generate branch name and PR details
	branchName := GenerateBranchName(event)
	prTitle := GeneratePRTitle(event)
	prDescription := GeneratePRDescription(event, fixResponse)

	// Create file changes
	changes := []FileChange{
		{
			FilePath: event.SourceFile,
			Content:  fixResponse.ProposedFix,
		},
	}

	// Create PR request
	prRequest := PRRequest{
		BranchName:  branchName,
		Title:       prTitle,
		Description: prDescription,
		Changes:     changes,
	}

	// Execute Git operations with retry logic
	err := w.healer.retryManager.ExecuteWithRetry(gitCtx, fmt.Sprintf("git-pr-%s", event.ID), func() error {
		return w.healer.gitClient.CreatePullRequest(gitCtx, prRequest)
	})

	if err != nil {
		// Check if it's a timeout or cancellation
		if ctx.Err() != nil {
			return fmt.Errorf("Git processing cancelled: %w", ctx.Err())
		}

		// Log the failure but don't fail the entire processing
		if w.logger != nil {
			w.logger.Error("Worker %d failed to create PR for event %s: %v", w.id, event.ID, err)
		}
		return fmt.Errorf("Git PR creation failed: %w", err)
	}

	if w.logger != nil {
		w.logger.Info("Worker %d successfully created PR for event %s: %s", w.id, event.ID, prTitle)
	}

	return nil
}

// extractSourceCode attempts to extract relevant source code context from the panic event
func (w *BackgroundWorker) extractSourceCode(event PanicEvent) string {
	// For now, we'll use the stack trace as context
	// In a more sophisticated implementation, we could:
	// 1. Read the actual source file at the panic location
	// 2. Extract surrounding lines of code
	// 3. Include relevant function definitions

	if event.SourceFile == "" || event.LineNumber == 0 {
		return ""
	}

	// Create a simple source context description
	return fmt.Sprintf("// Error occurred in file: %s at line %d in function: %s\n// Stack trace provides additional context",
		event.SourceFile, event.LineNumber, event.Function)
}

// storeFixResponse stores the AI fix response for later use by Git processing
func (w *BackgroundWorker) storeFixResponse(event PanicEvent, fixResponse *FixResponse) {
	// For now, just log the fix response
	// In a more complete implementation, this could:
	// 1. Store in a database or cache
	// 2. Queue for Git processing
	// 3. Notify other components

	if w.logger != nil {
		w.logger.Debug("Storing fix response for event %s: confidence=%.2f, valid=%v",
			event.ID, fixResponse.Confidence, fixResponse.IsValid)

		if fixResponse.ProposedFix != "" {
			w.logger.Debug("Proposed fix for event %s:\n%s", event.ID, fixResponse.ProposedFix)
		}

		if fixResponse.Explanation != "" {
			w.logger.Debug("Fix explanation for event %s: %s", event.ID, fixResponse.Explanation)
		}
	}
}

// WorkerPool manages a pool of background workers
type WorkerPool struct {
	workers []*BackgroundWorker
	healer  *Healer
	logger  Logger
	wg      sync.WaitGroup
	ctx     context.Context
	cancel  context.CancelFunc
	mu      sync.RWMutex
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(healer *Healer, logger Logger) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())

	return &WorkerPool{
		healer: healer,
		logger: logger,
		ctx:    ctx,
		cancel: cancel,
	}
}

// Start initializes and starts all workers in the pool
func (wp *WorkerPool) Start() error {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	if len(wp.workers) > 0 {
		return nil // Already started
	}

	// Create workers based on configuration
	workerCount := wp.healer.config.WorkerCount
	wp.workers = make([]*BackgroundWorker, workerCount)

	for i := 0; i < workerCount; i++ {
		worker := NewBackgroundWorker(i+1, wp.healer, wp.logger, &wp.wg)
		wp.workers[i] = worker

		if err := worker.Start(wp.ctx); err != nil {
			// Stop any workers that were already started
			wp.stopWorkers()
			return err
		}
	}

	if wp.logger != nil {
		wp.logger.Info("Worker pool started with %d workers", workerCount)
	}

	return nil
}

// Stop gracefully stops all workers in the pool
func (wp *WorkerPool) Stop() error {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	if len(wp.workers) == 0 {
		return nil // Already stopped
	}

	if wp.logger != nil {
		wp.logger.Info("Stopping worker pool with %d workers", len(wp.workers))
	}

	// Cancel context to signal all workers to stop
	wp.cancel()

	// Stop all workers
	wp.stopWorkers()

	// Wait for all workers to finish with timeout
	done := make(chan struct{})
	go func() {
		wp.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		if wp.logger != nil {
			wp.logger.Info("All workers stopped gracefully")
		}
	case <-time.After(30 * time.Second):
		if wp.logger != nil {
			wp.logger.Warn("Timeout waiting for workers to stop")
		}
	}

	// Clear workers slice
	wp.workers = nil

	return nil
}

// stopWorkers stops all workers without waiting
func (wp *WorkerPool) stopWorkers() {
	for _, worker := range wp.workers {
		if err := worker.Stop(); err != nil && wp.logger != nil {
			wp.logger.Error("Error stopping worker %d: %v", worker.id, err)
		}
	}
}

// GetWorkerCount returns the number of active workers
func (wp *WorkerPool) GetWorkerCount() int {
	wp.mu.RLock()
	defer wp.mu.RUnlock()
	return len(wp.workers)
}

// IsRunning returns true if the worker pool is running
func (wp *WorkerPool) IsRunning() bool {
	wp.mu.RLock()
	defer wp.mu.RUnlock()
	return len(wp.workers) > 0
}

// processEventAsync processes an event asynchronously with proper timeout management
func (w *BackgroundWorker) processEventAsync(ctx context.Context, event PanicEvent) {
	// Create a goroutine for async processing with timeout
	go func() {
		// Create a timeout context for the entire async operation
		asyncCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		// Combine with the original context to respect cancellation
		combinedCtx, combinedCancel := context.WithCancel(asyncCtx)
		defer combinedCancel()

		// Monitor original context cancellation
		go func() {
			select {
			case <-ctx.Done():
				combinedCancel()
			case <-asyncCtx.Done():
				// Timeout or completion
			}
		}()

		if w.logger != nil {
			w.logger.Debug("Worker %d starting async processing for event %s", w.id, event.ID)
		}

		// Process the event with full error handling
		err := w.processEventWithRetry(combinedCtx, event)
		if err != nil {
			if w.logger != nil {
				w.logger.Error("Worker %d async processing failed for event %s: %v", w.id, event.ID, err)
			}
		} else {
			if w.logger != nil {
				w.logger.Info("Worker %d async processing completed for event %s", w.id, event.ID)
			}
		}
	}()
}

// processEventWithTimeoutManagement adds additional timeout management for AI and Git operations
func (w *BackgroundWorker) processEventWithTimeoutManagement(ctx context.Context, event PanicEvent) error {
	// Store fix response for Git processing
	var fixResponse *FixResponse

	// Create multiple timeout contexts for different phases
	phases := []struct {
		name    string
		timeout time.Duration
		fn      func(context.Context) error
	}{
		{
			name:    "ai-processing",
			timeout: 45 * time.Second,
			fn: func(phaseCtx context.Context) error {
				var err error
				fixResponse, err = w.processEventWithAI(phaseCtx, event)
				return err
			},
		},
		{
			name:    "git-processing",
			timeout: 60 * time.Second,
			fn: func(phaseCtx context.Context) error {
				return w.processEventWithGit(phaseCtx, event, fixResponse)
			},
		},
	}

	for _, phase := range phases {
		phaseCtx, cancel := context.WithTimeout(ctx, phase.timeout)

		if w.logger != nil {
			w.logger.Debug("Worker %d starting phase '%s' for event %s (timeout: %v)",
				w.id, phase.name, event.ID, phase.timeout)
		}

		err := phase.fn(phaseCtx)
		cancel()

		if err != nil {
			if phaseCtx.Err() == context.DeadlineExceeded {
				return fmt.Errorf("phase '%s' timed out after %v: %w", phase.name, phase.timeout, err)
			}
			return fmt.Errorf("phase '%s' failed: %w", phase.name, err)
		}

		if w.logger != nil {
			w.logger.Debug("Worker %d completed phase '%s' for event %s", w.id, phase.name, event.ID)
		}
	}

	return nil
}
