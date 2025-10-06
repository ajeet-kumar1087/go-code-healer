// Package main demonstrates Go Code Healer integration in a microservice.
//
// This example shows how to use the healer in a microservice architecture
// with message queues, background workers, and service-to-service communication.
package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	healer "github.com/ajeet-kumar1087/go-code-healer"
)

// Message represents a message in our system
type Message struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Payload   map[string]interface{} `json:"payload"`
	Timestamp time.Time              `json:"timestamp"`
	Retries   int                    `json:"retries"`
}

// Service represents our microservice
type Service struct {
	name         string
	healer       *healer.Healer
	messageQueue chan Message
	workers      []*Worker
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
}

// Worker represents a background worker
type Worker struct {
	id      int
	service *Service
	ctx     context.Context
}

func main() {
	fmt.Println("Go Code Healer - Microservice Example")
	fmt.Println("=====================================")

	// Initialize the service
	service, err := initializeService("order-processor")
	if err != nil {
		log.Fatal("Failed to initialize service:", err)
	}

	// Start the service
	if err := service.Start(); err != nil {
		log.Fatal("Failed to start service:", err)
	}

	// Simulate incoming messages
	go service.simulateIncomingMessages()

	// Wait for shutdown signal
	service.waitForShutdown()
}

// initializeService creates and configures the microservice
func initializeService(name string) (*Service, error) {
	fmt.Printf("Initializing service: %s\n", name)

	// Configure the healer
	config := healer.Config{
		OpenAIAPIKey: getEnvOrDefault("HEALER_OPENAI_API_KEY", ""),
		GitHubToken:  getEnvOrDefault("HEALER_GITHUB_TOKEN", ""),
		RepoOwner:    getEnvOrDefault("HEALER_REPO_OWNER", ""),
		RepoName:     getEnvOrDefault("HEALER_REPO_NAME", ""),
		Enabled:      true,
		LogLevel:     "info",
		MaxQueueSize: 200, // Larger queue for microservice
		WorkerCount:  3,   // More workers for background processing
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

	// Create service context
	ctx, cancel := context.WithCancel(context.Background())

	service := &Service{
		name:         name,
		healer:       h,
		messageQueue: make(chan Message, 1000), // Large buffer for high throughput
		ctx:          ctx,
		cancel:       cancel,
	}

	return service, nil
}

// Start starts the microservice and its workers
func (s *Service) Start() error {
	fmt.Printf("Starting service: %s\n", s.name)

	// Start message processing workers
	workerCount := 5
	s.workers = make([]*Worker, workerCount)

	for i := 0; i < workerCount; i++ {
		worker := &Worker{
			id:      i + 1,
			service: s,
			ctx:     s.ctx,
		}
		s.workers[i] = worker

		s.wg.Add(1)
		healer.SafeGoroutine(func() {
			defer s.wg.Done()
			worker.run()
		})
	}

	// Start health check worker
	s.wg.Add(1)
	healer.SafeGoroutine(func() {
		defer s.wg.Done()
		s.healthCheckWorker()
	})

	// Start metrics collector
	s.wg.Add(1)
	healer.SafeGoroutine(func() {
		defer s.wg.Done()
		s.metricsCollector()
	})

	fmt.Printf("✓ Started %d workers for service: %s\n", workerCount, s.name)
	return nil
}

// Stop gracefully stops the microservice
func (s *Service) Stop() error {
	fmt.Printf("Stopping service: %s\n", s.name)

	// Cancel context to signal workers to stop
	s.cancel()

	// Close message queue
	close(s.messageQueue)

	// Wait for all workers to finish
	s.wg.Wait()

	// Stop healer
	if s.healer != nil {
		s.healer.Stop()
	}

	fmt.Printf("✓ Service stopped: %s\n", s.name)
	return nil
}

// Worker implementation

func (w *Worker) run() {
	fmt.Printf("Worker %d started\n", w.id)
	defer fmt.Printf("Worker %d stopped\n", w.id)

	for {
		select {
		case <-w.ctx.Done():
			return
		case message, ok := <-w.service.messageQueue:
			if !ok {
				return // Channel closed
			}
			w.processMessage(message)
		}
	}
}

func (w *Worker) processMessage(message Message) {
	// Add panic protection for message processing
	defer func() {
		if r := recover(); r != nil {
			// Let healer capture the panic
			if healer.IsGlobalHealerInstalled() {
				healer.RecoverAndHandle()
			}

			// Handle failed message
			w.handleFailedMessage(message, fmt.Errorf("panic: %v", r))
		}
	}()

	fmt.Printf("Worker %d processing message %s (type: %s)\n", w.id, message.ID, message.Type)

	// Simulate processing time
	time.Sleep(time.Duration(rand.Intn(500)) * time.Millisecond)

	// Process different message types
	switch message.Type {
	case "order.created":
		w.processOrderCreated(message)
	case "order.updated":
		w.processOrderUpdated(message)
	case "order.cancelled":
		w.processOrderCancelled(message)
	case "payment.processed":
		w.processPaymentProcessed(message)
	case "inventory.updated":
		w.processInventoryUpdated(message)
	default:
		panic(fmt.Sprintf("unknown message type: %s", message.Type))
	}

	fmt.Printf("Worker %d completed message %s\n", w.id, message.ID)
}

func (w *Worker) processOrderCreated(message Message) {
	// Simulate order processing that might panic
	orderID, ok := message.Payload["order_id"].(string)
	if !ok {
		panic("invalid order_id in message payload")
	}

	// Simulate database operations
	if orderID == "panic-order" {
		panic("database connection failed")
	}

	// Simulate external service calls
	if rand.Float32() < 0.1 {
		panic("external payment service unavailable")
	}

	// Simulate inventory check
	w.checkInventory(message.Payload)

	// Simulate order validation
	w.validateOrder(message.Payload)

	fmt.Printf("  Order created: %s\n", orderID)
}

func (w *Worker) processOrderUpdated(message Message) {
	orderID := message.Payload["order_id"].(string)

	// Simulate update processing that might panic
	if orderID == "invalid-update" {
		var nilMap map[string]interface{}
		nilMap["key"] = "value" // panic: assignment to entry in nil map
	}

	fmt.Printf("  Order updated: %s\n", orderID)
}

func (w *Worker) processOrderCancelled(message Message) {
	orderID := message.Payload["order_id"].(string)

	// Simulate cancellation processing
	if orderID == "cannot-cancel" {
		panic("order cannot be cancelled: already shipped")
	}

	fmt.Printf("  Order cancelled: %s\n", orderID)
}

func (w *Worker) processPaymentProcessed(message Message) {
	paymentID := message.Payload["payment_id"].(string)

	// Simulate payment processing
	if paymentID == "failed-payment" {
		slice := []string{"a", "b", "c"}
		fmt.Println(slice[10]) // panic: index out of range
	}

	fmt.Printf("  Payment processed: %s\n", paymentID)
}

func (w *Worker) processInventoryUpdated(message Message) {
	productID := message.Payload["product_id"].(string)

	// Simulate inventory update
	if productID == "invalid-product" {
		var i interface{} = "not-a-number"
		quantity := i.(int) // panic: interface conversion
		fmt.Printf("Quantity: %d\n", quantity)
	}

	fmt.Printf("  Inventory updated: %s\n", productID)
}

func (w *Worker) checkInventory(payload map[string]interface{}) {
	// Simulate inventory check that might panic
	if rand.Float32() < 0.05 {
		panic("inventory service timeout")
	}
}

func (w *Worker) validateOrder(payload map[string]interface{}) {
	// Simulate order validation that might panic
	if customerID, ok := payload["customer_id"]; !ok || customerID == nil {
		panic("missing customer_id in order")
	}
}

func (w *Worker) handleFailedMessage(message Message, err error) {
	fmt.Printf("Worker %d failed to process message %s: %v\n", w.id, message.ID, err)

	// Implement retry logic
	message.Retries++
	if message.Retries < 3 {
		fmt.Printf("  Retrying message %s (attempt %d)\n", message.ID, message.Retries+1)

		// Re-queue the message with delay
		go func() {
			time.Sleep(time.Duration(message.Retries) * time.Second)
			select {
			case w.service.messageQueue <- message:
				// Message re-queued successfully
			default:
				fmt.Printf("  Failed to re-queue message %s: queue full\n", message.ID)
			}
		}()
	} else {
		fmt.Printf("  Message %s exceeded max retries, sending to dead letter queue\n", message.ID)
		w.sendToDeadLetterQueue(message)
	}
}

func (w *Worker) sendToDeadLetterQueue(message Message) {
	// In a real implementation, this would send to a dead letter queue
	fmt.Printf("  Dead letter: %s\n", message.ID)
}

// Background workers

func (s *Service) healthCheckWorker() {
	fmt.Println("Health check worker started")
	defer fmt.Println("Health check worker stopped")

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.performHealthCheck()
		}
	}
}

func (s *Service) performHealthCheck() {
	defer func() {
		if r := recover(); r != nil {
			if healer.IsGlobalHealerInstalled() {
				healer.RecoverAndHandle()
			}
			fmt.Printf("Health check failed: %v\n", r)
		}
	}()

	// Simulate health check that might panic
	if rand.Float32() < 0.05 {
		panic("health check service unavailable")
	}

	// Check healer status
	if s.healer != nil {
		status := s.healer.GetStatus()
		queueStats := s.healer.GetQueueStats()

		fmt.Printf("Health check: Service healthy, Healer queue: %v/%v\n",
			queueStats["queue_length"], queueStats["queue_capacity"])

		// Alert if queue is getting full
		if queueLength := queueStats["queue_length"].(int); queueLength > 150 {
			fmt.Printf("WARNING: Healer queue is %d%% full\n",
				(queueLength*100)/queueStats["queue_capacity"].(int))
		}

		_ = status // Use status if needed
	}
}

func (s *Service) metricsCollector() {
	fmt.Println("Metrics collector started")
	defer fmt.Println("Metrics collector stopped")

	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.collectMetrics()
		}
	}
}

func (s *Service) collectMetrics() {
	defer func() {
		if r := recover(); r != nil {
			if healer.IsGlobalHealerInstalled() {
				healer.RecoverAndHandle()
			}
			fmt.Printf("Metrics collection failed: %v\n", r)
		}
	}()

	// Simulate metrics collection that might panic
	if rand.Float32() < 0.03 {
		panic("metrics database connection failed")
	}

	// Collect service metrics
	queueLength := len(s.messageQueue)
	fmt.Printf("Metrics: Queue length: %d, Workers: %d\n", queueLength, len(s.workers))

	// Collect healer metrics if available
	if s.healer != nil {
		healerStats := s.healer.GetQueueStats()
		fmt.Printf("Metrics: Healer queue: %v, Dropped events: %v\n",
			healerStats["queue_length"], healerStats["dropped_events"])
	}
}

// Message simulation

func (s *Service) simulateIncomingMessages() {
	fmt.Println("Starting message simulation...")

	messageTypes := []string{
		"order.created",
		"order.updated",
		"order.cancelled",
		"payment.processed",
		"inventory.updated",
	}

	// Generate messages at regular intervals
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	messageCounter := 0
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			messageCounter++

			// Create a message
			message := Message{
				ID:        fmt.Sprintf("msg-%d", messageCounter),
				Type:      messageTypes[rand.Intn(len(messageTypes))],
				Timestamp: time.Now(),
				Payload:   s.generateMessagePayload(messageCounter),
			}

			// Send to queue
			select {
			case s.messageQueue <- message:
				fmt.Printf("Generated message: %s (%s)\n", message.ID, message.Type)
			default:
				fmt.Printf("Message queue full, dropping message: %s\n", message.ID)
			}
		}
	}
}

func (s *Service) generateMessagePayload(counter int) map[string]interface{} {
	payload := make(map[string]interface{})

	// Add some test data that might cause panics
	switch counter % 20 {
	case 5:
		payload["order_id"] = "panic-order"
	case 10:
		payload["order_id"] = "invalid-update"
	case 15:
		payload["payment_id"] = "failed-payment"
	default:
		payload["order_id"] = fmt.Sprintf("order-%d", counter)
		payload["customer_id"] = fmt.Sprintf("customer-%d", counter%100)
		payload["product_id"] = fmt.Sprintf("product-%d", counter%50)
		payload["payment_id"] = fmt.Sprintf("payment-%d", counter)
	}

	payload["amount"] = rand.Float64() * 1000
	payload["quantity"] = rand.Intn(10) + 1

	return payload
}

// Utility functions

func (s *Service) waitForShutdown() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("\nReceived shutdown signal...")
	s.Stop()
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
