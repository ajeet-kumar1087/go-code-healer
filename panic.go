package healer

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"runtime"
	"strings"
	"time"
)

// PanicEvent represents a captured panic with context
type PanicEvent struct {
	ID          string     `json:"id"`
	Timestamp   time.Time  `json:"timestamp"`
	Error       string     `json:"error"`
	StackTrace  string     `json:"stack_trace"`
	SourceFile  string     `json:"source_file"`
	LineNumber  int        `json:"line_number"`
	Function    string     `json:"function"`
	ProcessedAt *time.Time `json:"processed_at,omitempty"`
	Status      string     `json:"status"` // "queued", "processing", "completed", "failed"
}

// NewPanicEvent creates a new PanicEvent from a panic value
func NewPanicEvent(panicValue any) *PanicEvent {
	event := &PanicEvent{
		ID:        generateID(),
		Timestamp: time.Now(),
		Error:     fmt.Sprintf("%v", panicValue),
		Status:    "queued",
	}

	// Extract stack trace and source location
	event.extractStackTrace()
	return event
}

// extractStackTrace captures the current stack trace and extracts source location
func (pe *PanicEvent) extractStackTrace() {
	// Get stack trace with up to 32 frames, skipping the first 3 frames
	// (runtime.Callers, extractStackTrace, NewPanicEvent)
	pc := make([]uintptr, 32)
	n := runtime.Callers(3, pc)
	pc = pc[:n]

	frames := runtime.CallersFrames(pc)
	var stackLines []string
	var firstUserFrame *runtime.Frame

	for {
		frame, more := frames.Next()

		// Skip runtime and healer package frames to find the first user frame
		if firstUserFrame == nil && !strings.Contains(frame.File, "runtime/") &&
			!strings.Contains(frame.File, "/healer/") {
			firstUserFrame = &frame
		}

		// Format stack trace line
		stackLine := fmt.Sprintf("%s:%d %s", frame.File, frame.Line, frame.Function)
		stackLines = append(stackLines, stackLine)

		if !more {
			break
		}
	}

	pe.StackTrace = strings.Join(stackLines, "\n")

	// Set source location from the first user frame
	if firstUserFrame != nil {
		pe.SourceFile = firstUserFrame.File
		pe.LineNumber = firstUserFrame.Line
		pe.Function = firstUserFrame.Function
	}
}

// ToJSON serializes the PanicEvent to JSON for logging and API calls
func (pe *PanicEvent) ToJSON() ([]byte, error) {
	return json.Marshal(pe)
}

// ToJSONString serializes the PanicEvent to a JSON string
func (pe *PanicEvent) ToJSONString() (string, error) {
	data, err := pe.ToJSON()
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// GetSummary returns a brief summary of the panic event
func (pe *PanicEvent) GetSummary() string {
	return fmt.Sprintf("Panic at %s:%d in %s: %s",
		pe.SourceFile, pe.LineNumber, pe.Function, pe.Error)
}

// GetContext returns contextual information about the panic for AI processing
func (pe *PanicEvent) GetContext() string {
	var context strings.Builder

	context.WriteString(fmt.Sprintf("Error: %s\n", pe.Error))
	context.WriteString(fmt.Sprintf("Location: %s:%d\n", pe.SourceFile, pe.LineNumber))
	context.WriteString(fmt.Sprintf("Function: %s\n", pe.Function))
	context.WriteString(fmt.Sprintf("Timestamp: %s\n", pe.Timestamp.Format(time.RFC3339)))
	context.WriteString("Stack Trace:\n")
	context.WriteString(pe.StackTrace)

	return context.String()
}

// generateID creates a unique identifier for the panic event
func generateID() string {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based ID if random generation fails
		return fmt.Sprintf("panic_%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)
}

// ProcessingResult represents the result of processing a panic event
type ProcessingResult struct {
	PanicID     string    `json:"panic_id"`
	Success     bool      `json:"success"`
	PRUrl       string    `json:"pr_url,omitempty"`
	Error       string    `json:"error,omitempty"`
	ProcessedAt time.Time `json:"processed_at"`
}

// PanicCapture handles the interception of panics
type PanicCapture struct {
	healer HealerInterface
	logger LoggerInterface
}

// HealerInterface defines the interface for the healer
type HealerInterface interface {
	GetQueueManager() QueueManagerInterface
	GetErrorQueue() chan PanicEvent
}

// QueueManagerInterface defines the interface for queue management
type QueueManagerInterface interface {
	EnqueueEvent(event PanicEvent) bool
}

// NewPanicCapture creates a new PanicCapture instance
func NewPanicCapture(healer HealerInterface, logger LoggerInterface) *PanicCapture {
	return &PanicCapture{
		healer: healer,
		logger: logger,
	}
}

// InstallHandler installs the panic handler, preserving any existing handler
func (pc *PanicCapture) InstallHandler() {
	// Store the original panic handler if one exists
	// Note: Go doesn't provide a way to get the current panic handler,
	// so we'll work with the assumption that we're the first to install one
	// or that we're replacing a previous healer installation

	// Set up our panic handler by ensuring the global healer is available
	pc.setupPanicHandler()

	// Log installation instructions for users
	if pc.logger != nil {
		pc.logger.Info("Panic handler installed. To capture panics, use one of these approaches:")
		pc.logger.Info("1. Add 'defer healer.HandlePanic()' to functions that might panic")
		pc.logger.Info("2. Add 'defer healer.RecoverAndHandle()' for graceful recovery without re-panicking")
		pc.logger.Info("3. Use healer.WrapFunction() or healer.WrapFunctionWithRecovery() to wrap functions")
	}
}

// setupPanicHandler sets up the panic recovery mechanism
func (pc *PanicCapture) setupPanicHandler() {
	// The actual panic interception happens in CapturePanic when called from
	// HandlePanic() or RecoverAndHandle() functions
}

// CapturePanic processes a panic and queues it for background processing
func (pc *PanicCapture) CapturePanic(panicValue any) {
	// Create panic event immediately
	event := NewPanicEvent(panicValue)

	// Log the panic immediately for debugging
	if pc.logger != nil {
		pc.logger.Error("Panic captured: %s", event.GetSummary())
		pc.logger.Debug("Panic details: %s", event.GetContext())
	}

	// Queue the event for background processing using queue manager
	if pc.healer != nil && pc.healer.GetQueueManager() != nil {
		success := pc.healer.GetQueueManager().EnqueueEvent(*event)
		if !success && pc.logger != nil {
			pc.logger.Error("Failed to enqueue panic event: %s", event.ID)
		}
	} else {
		// Fallback to direct queue access if queue manager is not available
		if pc.healer != nil && pc.healer.GetErrorQueue() != nil {
			select {
			case pc.healer.GetErrorQueue() <- *event:
				if pc.logger != nil {
					pc.logger.Debug("Panic event queued for processing: %s", event.ID)
				}
			default:
				// Queue is full, log the issue but don't block
				if pc.logger != nil {
					pc.logger.Warn("Panic event queue is full, dropping event: %s", event.ID)
				}
			}
		}
	}

	// Immediately return control to allow existing panic recovery mechanisms to work
	// This ensures we don't interfere with the application's normal panic handling
}
