package main

import (
	"context"
	"log"
	"time"

	"github.com/ajeet-kumar1087/go-code-healer/ai"
)

// SimpleLogger implements a basic logger for the demo
type SimpleLogger struct{}

func (l *SimpleLogger) Debug(msg string, args ...any) { log.Printf("[DEBUG] "+msg, args...) }
func (l *SimpleLogger) Info(msg string, args ...any)  { log.Printf("[INFO] "+msg, args...) }
func (l *SimpleLogger) Warn(msg string, args ...any)  { log.Printf("[WARN] "+msg, args...) }
func (l *SimpleLogger) Error(msg string, args ...any) { log.Printf("[ERROR] "+msg, args...) }
func (l *SimpleLogger) SetLevel(level interface{})    { /* no-op for demo */ }

// MockGitClient implements GitClientInterface for demonstration
type MockGitClient struct{}

func (m *MockGitClient) CreatePullRequest(ctx context.Context, request ai.PRRequest) error {
	log.Printf("Creating PR: %s", request.Title)
	log.Printf("Branch: %s", request.BranchName)
	log.Printf("Files changed: %d", len(request.Changes))
	log.Printf("Description preview: %.100s...", request.Description)
	return nil
}

func main() {
	log.Println("=== AI Session-Based Error Handling Demo ===")
	log.Println("This demo shows how to use the AI session manager directly")

	// Create logger
	logger := &SimpleLogger{}

	// Create mock Git client
	gitClient := &MockGitClient{}

	// Create a simple OpenAI client for demonstration (nil logger for simplicity)
	openaiClient := ai.NewOpenAIClient("demo-key", "gpt-4", nil)

	// Create session manager directly
	session := ai.NewSessionManager(openaiClient, nil, gitClient, nil)

	// Simulate a nil pointer error
	simulateNilPointerError(session, logger)
}

func simulateNilPointerError(session *ai.SessionManager, logger *SimpleLogger) {
	log.Println("\n--- Simulating Nil Pointer Error ---")

	// Prepare error information
	errorInfo := &ai.ErrorInfo{
		Error: "runtime error: invalid memory address or nil pointer dereference",
		StackTrace: `panic: runtime error: invalid memory address or nil pointer dereference

goroutine 1 [running]:
main.processUser(0x0)
	/app/user.go:45 +0x1a
main.main()
	/app/main.go:12 +0x29`,
		SourceFile: "user.go",
		LineNumber: 45,
		Function:   "processUser",
		Timestamp:  time.Now(),
		Severity:   "critical",
	}

	// Prepare code context
	codeContext := &ai.CodeContext{
		SourceCode: `func processUser(user *User) {
	// This line causes nil pointer dereference
	fmt.Printf("Processing user: %s", user.Name)
	
	// Additional processing
	user.LastAccessed = time.Now()
}`,
		RelatedFiles: []string{"main.go", "types.go"},
		ImportedPkgs: []string{"fmt", "time"},
		FunctionSig:  "processUser(user *User)",
		StructDefs:   []string{"type User struct { Name string; LastAccessed time.Time }"},
	}

	// Initiate session (this will fail with demo key, but shows the structure)
	ctx := context.Background()
	result, err := session.InitiateSession(ctx, errorInfo, codeContext)
	if err != nil {
		log.Printf("Session failed (expected with demo key): %v", err)
		log.Printf("In a real scenario with valid API keys, this would:")
		log.Printf("1. Analyze the error with AI")
		log.Printf("2. Generate a comprehensive fix")
		log.Printf("3. Create a pull request with the solution")
		return
	}

	log.Printf("Session completed successfully!")
	log.Printf("Session ID: %s", result.SessionID)
	log.Printf("Duration: %v", result.Duration)
	log.Printf("AI Provider: %s", result.FixResponse.Provider)
	log.Printf("Confidence: %.2f", result.FixResponse.Confidence)
	log.Printf("PR Created: %s", result.PRResult.BranchName)
}
