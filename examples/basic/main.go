// Package main demonstrates basic usage of the Go Code Healer.
//
// This example shows how to integrate the healer into a simple application
// with various types of functions that might panic.
package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	healer "github.com/ajeet-kumar1087/go-code-healer"
)

func main() {
	fmt.Println("Go Code Healer - Basic Example")
	fmt.Println("==============================")

	// Configure the healer
	config := healer.Config{
		OpenAIAPIKey: getEnvOrDefault("HEALER_OPENAI_API_KEY", ""),
		GitHubToken:  getEnvOrDefault("HEALER_GITHUB_TOKEN", ""),
		RepoOwner:    getEnvOrDefault("HEALER_REPO_OWNER", ""),
		RepoName:     getEnvOrDefault("HEALER_REPO_NAME", ""),
		Enabled:      true,
		LogLevel:     "info",
	}

	// Install global panic handler
	h, err := healer.InstallGlobalPanicHandler(config)
	if err != nil {
		log.Printf("Failed to install healer: %v", err)
		log.Println("Continuing without healer...")
		runExampleWithoutHealer()
		return
	}
	defer h.Stop()

	fmt.Println("✓ Healer installed successfully")
	fmt.Println()

	// Show healer status
	showHealerStatus(h)

	// Run examples
	runBasicExamples()
	runAdvancedExamples()
	runBackgroundWorkerExample()

	fmt.Println("\nExample completed. Check your GitHub repository for any pull requests!")
}

// getEnvOrDefault returns environment variable value or default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// showHealerStatus displays current healer status
func showHealerStatus(h *healer.Healer) {
	status := h.GetStatus()
	queueStats := h.GetQueueStats()

	fmt.Printf("Healer Status:\n")
	fmt.Printf("  Enabled: %v\n", status["enabled"])
	fmt.Printf("  Running: %v\n", status["running"])
	fmt.Printf("  Queue Capacity: %v\n", queueStats["queue_capacity"])
	fmt.Printf("  Worker Count: %v\n", queueStats["worker_count"])
	fmt.Println()
}

// runBasicExamples demonstrates basic panic capture patterns
func runBasicExamples() {
	fmt.Println("Running Basic Examples:")
	fmt.Println("-----------------------")

	// Example 1: Function with manual panic capture (re-panics)
	fmt.Println("1. Testing function with HandlePanic() - will capture and re-panic")
	func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("   Caught panic: %v\n", r)
			}
		}()

		func() {
			defer healer.HandlePanic() // Captures panic and re-panics
			causeNilPointerPanic()
		}()
	}()

	// Example 2: Function with graceful recovery
	fmt.Println("2. Testing function with RecoverAndHandle() - will capture and recover")
	func() {
		defer healer.RecoverAndHandle() // Captures panic and recovers gracefully
		causeIndexOutOfBoundsPanic()
	}()
	fmt.Println("   Function continued after panic recovery")

	// Example 3: Wrapped function
	fmt.Println("3. Testing wrapped function")
	safeFunction := healer.WrapFunctionWithRecovery(func() {
		causeMapPanic()
	})
	safeFunction()
	fmt.Println("   Wrapped function completed")

	fmt.Println()
}

// runAdvancedExamples demonstrates advanced usage patterns
func runAdvancedExamples() {
	fmt.Println("Running Advanced Examples:")
	fmt.Println("--------------------------")

	// Example 1: Safe goroutines
	fmt.Println("1. Testing safe goroutines")
	done := make(chan bool, 3)

	for i := 0; i < 3; i++ {
		i := i // Capture loop variable
		healer.SafeGoroutine(func() {
			defer func() { done <- true }()
			fmt.Printf("   Goroutine %d starting\n", i)

			if i == 1 {
				// This goroutine will panic but be handled gracefully
				causeSlicePanic()
			}

			time.Sleep(100 * time.Millisecond)
			fmt.Printf("   Goroutine %d completed\n", i)
		})
	}

	// Wait for all goroutines to complete
	for i := 0; i < 3; i++ {
		<-done
	}
	fmt.Println("   All goroutines completed")

	// Example 2: Function with arguments
	fmt.Println("2. Testing function with arguments")
	safeVariadicFunc := healer.WrapFunctionWithArgsAndRecovery(func(args ...any) {
		fmt.Printf("   Processing args: %v\n", args)
		if len(args) > 2 {
			causeInterfacePanic()
		}
		fmt.Println("   Arguments processed successfully")
	})

	safeVariadicFunc("hello", "world")        // Should work fine
	safeVariadicFunc("this", "will", "panic") // Should panic but recover

	fmt.Println()
}

// runBackgroundWorkerExample demonstrates background worker with panic protection
func runBackgroundWorkerExample() {
	fmt.Println("Running Background Worker Example:")
	fmt.Println("----------------------------------")

	// Start a background worker that processes jobs
	jobQueue := make(chan int, 10)
	workerDone := make(chan bool)

	// Add some jobs to the queue
	for i := 1; i <= 5; i++ {
		jobQueue <- i
	}
	close(jobQueue)

	// Start worker with panic protection
	healer.SafeGoroutine(func() {
		defer func() { workerDone <- true }()

		fmt.Println("   Background worker started")
		for job := range jobQueue {
			fmt.Printf("   Processing job %d\n", job)

			// Simulate work that might panic
			if job == 3 {
				fmt.Println("   Job 3 will cause a panic...")
				causeChannelPanic()
			}

			time.Sleep(50 * time.Millisecond)
			fmt.Printf("   Job %d completed\n", job)
		}
		fmt.Println("   Background worker finished")
	})

	// Wait for worker to complete
	<-workerDone
	fmt.Println("   Background processing completed")
	fmt.Println()
}

// runExampleWithoutHealer runs a simple example when healer is not available
func runExampleWithoutHealer() {
	fmt.Println("Running without healer (panics will not be captured)")
	fmt.Println("Note: In a real application, you might want to add basic panic recovery")

	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Caught panic without healer: %v\n", r)
		}
	}()

	// This will panic and be caught by the defer above
	causeNilPointerPanic()
}

// Example functions that cause different types of panics
// These represent common panic scenarios in Go applications

func causeNilPointerPanic() {
	fmt.Println("   Causing nil pointer panic...")
	var ptr *string
	fmt.Println(*ptr) // This will panic
}

func causeIndexOutOfBoundsPanic() {
	fmt.Println("   Causing index out of bounds panic...")
	slice := []int{1, 2, 3}
	fmt.Println(slice[10]) // This will panic
}

func causeMapPanic() {
	fmt.Println("   Causing map panic...")
	var m map[string]int
	m["key"] = 42 // This will panic (assignment to nil map)
}

func causeSlicePanic() {
	fmt.Println("   Causing slice panic...")
	slice := make([]int, 5)
	fmt.Println(slice[100]) // This will panic
}

func causeInterfacePanic() {
	fmt.Println("   Causing interface panic...")
	var i interface{} = "string"
	num := i.(int) // This will panic (type assertion failure)
	fmt.Println(num)
}

func causeChannelPanic() {
	fmt.Println("   Causing channel panic...")
	ch := make(chan int)
	close(ch)
	ch <- 42 // This will panic (send on closed channel)
}

// Additional utility functions for demonstration

func simulateRandomPanic() {
	panicTypes := []func(){
		causeNilPointerPanic,
		causeIndexOutOfBoundsPanic,
		causeMapPanic,
		causeSlicePanic,
		causeInterfacePanic,
	}

	// Randomly select a panic type
	panicFunc := panicTypes[rand.Intn(len(panicTypes))]
	panicFunc()
}

func demonstrateErrorHandling() {
	fmt.Println("Demonstrating error handling with panic recovery:")

	result, err := processDataSafely([]byte("test data"))
	if err != nil {
		fmt.Printf("   Error occurred: %v\n", err)
	} else {
		fmt.Printf("   Result: %s\n", result)
	}

	// This will cause a panic but return an error instead
	result2, err2 := processDataSafely(nil)
	if err2 != nil {
		fmt.Printf("   Error occurred: %v\n", err2)
	} else {
		fmt.Printf("   Result: %s\n", result2)
	}
}

func processDataSafely(data []byte) (result string, err error) {
	defer func() {
		if r := recover(); r != nil {
			// Let healer capture the panic if installed
			if healer.IsGlobalHealerInstalled() {
				healer.RecoverAndHandle()
			}
			// Convert panic to error
			err = fmt.Errorf("processing failed: %v", r)
		}
	}()

	// Simulate processing that might panic
	if data == nil {
		panic("nil data provided")
	}

	return fmt.Sprintf("processed: %s", string(data)), nil
}
