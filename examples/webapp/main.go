// Package main demonstrates Go Code Healer integration in a web application.
//
// This example shows how to use the healer with HTTP handlers, middleware,
// and background processing in a typical web application scenario.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	healer "github.com/ajeet-kumar1087/go-code-healer"
)

// Application represents our web application
type Application struct {
	healer *healer.Healer
	server *http.Server
}

// User represents a user in our system
type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// Response represents a standard API response
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func main() {
	fmt.Println("Go Code Healer - Web Application Example")
	fmt.Println("========================================")

	// Initialize the application
	app, err := initializeApplication()
	if err != nil {
		log.Fatal("Failed to initialize application:", err)
	}

	// Start background workers
	app.startBackgroundWorkers()

	// Setup HTTP routes
	app.setupRoutes()

	// Start the server
	go func() {
		fmt.Printf("Server starting on http://localhost:8080\n")
		fmt.Println("Try these endpoints:")
		fmt.Println("  GET  /health          - Health check")
		fmt.Println("  GET  /users           - List users (may panic)")
		fmt.Println("  GET  /users/{id}      - Get user by ID (may panic)")
		fmt.Println("  POST /users           - Create user (may panic)")
		fmt.Println("  GET  /panic           - Intentionally cause panic")
		fmt.Println("  GET  /healer/status   - Healer status")
		fmt.Println()

		if err := app.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Server failed to start:", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	app.waitForShutdown()
}

// initializeApplication sets up the application with healer
func initializeApplication() (*Application, error) {
	// Configure the healer
	config := healer.Config{
		OpenAIAPIKey: getEnvOrDefault("HEALER_OPENAI_API_KEY", ""),
		GitHubToken:  getEnvOrDefault("HEALER_GITHUB_TOKEN", ""),
		RepoOwner:    getEnvOrDefault("HEALER_REPO_OWNER", ""),
		RepoName:     getEnvOrDefault("HEALER_REPO_NAME", ""),
		Enabled:      true,
		LogLevel:     "info",
		MaxQueueSize: 50, // Smaller queue for web app
		WorkerCount:  1,  // Single worker for demo
	}

	// Install healer
	h, err := healer.InstallGlobalPanicHandler(config)
	if err != nil {
		log.Printf("Warning: Failed to install healer: %v", err)
		log.Println("Continuing without healer...")
		h = nil
	} else {
		fmt.Println("✓ Healer installed successfully")
	}

	// Create HTTP server
	server := &http.Server{
		Addr:         ":8080",
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return &Application{
		healer: h,
		server: server,
	}, nil
}

// setupRoutes configures HTTP routes with healer protection
func (app *Application) setupRoutes() {
	mux := http.NewServeMux()

	// Health check endpoint (no panic protection needed)
	mux.HandleFunc("/health", app.handleHealth)

	// API endpoints with panic protection
	mux.HandleFunc("/users", healer.WrapHTTPHandler(app.handleUsers))
	mux.HandleFunc("/users/", healer.WrapHTTPHandler(app.handleUserByID))
	mux.HandleFunc("/panic", healer.WrapHTTPHandler(app.handlePanic))

	// Healer status endpoint
	mux.HandleFunc("/healer/status", app.handleHealerStatus)

	// Add logging middleware
	app.server.Handler = loggingMiddleware(mux)
}

// startBackgroundWorkers starts background processing with panic protection
func (app *Application) startBackgroundWorkers() {
	// Start a background job processor
	healer.SafeGoroutine(func() {
		app.backgroundJobProcessor()
	})

	// Start a periodic cleanup worker
	healer.SafeGoroutine(func() {
		app.periodicCleanup()
	})

	fmt.Println("✓ Background workers started")
}

// HTTP Handlers

func (app *Application) handleHealth(w http.ResponseWriter, r *http.Request) {
	response := Response{
		Success: true,
		Data: map[string]interface{}{
			"status":    "healthy",
			"timestamp": time.Now().Unix(),
			"healer":    app.healer != nil,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (app *Application) handleUsers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		app.handleGetUsers(w, r)
	case http.MethodPost:
		app.handleCreateUser(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (app *Application) handleGetUsers(w http.ResponseWriter, r *http.Request) {
	// Simulate random panic (20% chance)
	if rand.Float32() < 0.2 {
		panic("database connection failed")
	}

	users := []User{
		{ID: 1, Name: "John Doe", Email: "john@example.com"},
		{ID: 2, Name: "Jane Smith", Email: "jane@example.com"},
		{ID: 3, Name: "Bob Johnson", Email: "bob@example.com"},
	}

	response := Response{
		Success: true,
		Data:    users,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (app *Application) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	var user User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Simulate validation panic
	if user.Name == "" {
		panic("user name cannot be empty")
	}

	// Simulate database panic
	if user.Email == "panic@example.com" {
		panic("database constraint violation")
	}

	// Assign ID and return created user
	user.ID = rand.Intn(1000) + 100

	response := Response{
		Success: true,
		Data:    user,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func (app *Application) handleUserByID(w http.ResponseWriter, r *http.Request) {
	// Extract ID from URL path
	path := r.URL.Path
	idStr := path[len("/users/"):]

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// Simulate various panic conditions
	switch id {
	case 999:
		panic("user not found in database")
	case 666:
		var nilPointer *User
		fmt.Println(nilPointer.Name) // nil pointer panic
	case 404:
		slice := []int{1, 2, 3}
		fmt.Println(slice[10]) // index out of bounds
	}

	user := User{
		ID:    id,
		Name:  fmt.Sprintf("User %d", id),
		Email: fmt.Sprintf("user%d@example.com", id),
	}

	response := Response{
		Success: true,
		Data:    user,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (app *Application) handlePanic(w http.ResponseWriter, r *http.Request) {
	panicType := r.URL.Query().Get("type")

	switch panicType {
	case "nil":
		var ptr *string
		fmt.Println(*ptr)
	case "slice":
		slice := []int{1, 2, 3}
		fmt.Println(slice[100])
	case "map":
		var m map[string]int
		m["key"] = 42
	case "interface":
		var i interface{} = "string"
		num := i.(int)
		fmt.Println(num)
	default:
		panic("intentional panic for testing")
	}
}

func (app *Application) handleHealerStatus(w http.ResponseWriter, r *http.Request) {
	var status map[string]interface{}

	if app.healer != nil {
		status = app.healer.GetStatus()
		queueStats := app.healer.GetQueueStats()
		status["queue_stats"] = queueStats
	} else {
		status = map[string]interface{}{
			"enabled": false,
			"message": "Healer not installed",
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// Background Workers

func (app *Application) backgroundJobProcessor() {
	fmt.Println("Background job processor started")

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			app.processJob()
		}
	}
}

func (app *Application) processJob() {
	fmt.Println("Processing background job...")

	// Simulate job processing that might panic
	if rand.Float32() < 0.3 {
		panic("background job failed: external service unavailable")
	}

	// Simulate work
	time.Sleep(100 * time.Millisecond)
	fmt.Println("Background job completed successfully")
}

func (app *Application) periodicCleanup() {
	fmt.Println("Periodic cleanup worker started")

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			app.performCleanup()
		}
	}
}

func (app *Application) performCleanup() {
	fmt.Println("Performing periodic cleanup...")

	// Simulate cleanup that might panic
	if rand.Float32() < 0.1 {
		panic("cleanup failed: disk full")
	}

	fmt.Println("Cleanup completed")
}

// Middleware

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap the response writer to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Process request with panic recovery
		func() {
			defer func() {
				if r := recover(); r != nil {
					// Let healer handle the panic if installed
					if healer.IsGlobalHealerInstalled() {
						healer.RecoverAndHandle()
					}

					// Return error response
					wrapped.WriteHeader(http.StatusInternalServerError)
					json.NewEncoder(wrapped).Encode(Response{
						Success: false,
						Error:   "Internal server error",
					})
				}
			}()

			next.ServeHTTP(wrapped, r)
		}()

		// Log the request
		duration := time.Since(start)
		log.Printf("%s %s %d %v", r.Method, r.URL.Path, wrapped.statusCode, duration)
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Utility functions

func (app *Application) waitForShutdown() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("\nShutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := app.server.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	if app.healer != nil {
		app.healer.Stop()
	}

	fmt.Println("Server exited")
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
