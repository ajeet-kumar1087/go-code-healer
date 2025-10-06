package internal

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
)

// Config represents the main configuration structure
// This is a copy of the main package Config to avoid circular imports
type Config struct {
	OpenAIAPIKey  string `json:"openai_api_key"`
	OpenAIModel   string `json:"openai_model,omitempty"`
	GitHubToken   string `json:"github_token"`
	RepoOwner     string `json:"repo_owner"`
	RepoName      string `json:"repo_name"`
	Enabled       bool   `json:"enabled"`
	MaxQueueSize  int    `json:"max_queue_size,omitempty"`
	WorkerCount   int    `json:"worker_count,omitempty"`
	RetryAttempts int    `json:"retry_attempts,omitempty"`
	LogLevel      string `json:"log_level,omitempty"`
}

// DefaultConfig returns a Config with default values
func DefaultConfig() Config {
	return Config{
		OpenAIModel:   "gpt-4",
		Enabled:       true,
		MaxQueueSize:  100,
		WorkerCount:   2,
		RetryAttempts: 3,
		LogLevel:      "info",
	}
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	var errs []error

	// Check required fields when enabled
	if c.Enabled {
		if c.OpenAIAPIKey == "" {
			errs = append(errs, errors.New("OpenAI API key is required when healer is enabled"))
		}

		if c.GitHubToken == "" {
			errs = append(errs, errors.New("GitHub token is required when healer is enabled"))
		}

		if c.RepoOwner == "" {
			errs = append(errs, errors.New("repository owner is required when healer is enabled"))
		}

		if c.RepoName == "" {
			errs = append(errs, errors.New("repository name is required when healer is enabled"))
		}
	}

	// Validate numeric fields
	if c.MaxQueueSize <= 0 {
		errs = append(errs, errors.New("max queue size must be greater than 0"))
	}

	if c.WorkerCount <= 0 {
		errs = append(errs, errors.New("worker count must be greater than 0"))
	}

	if c.RetryAttempts < 0 {
		errs = append(errs, errors.New("retry attempts cannot be negative"))
	}

	// Validate log level
	validLogLevels := []string{"debug", "info", "warn", "error"}
	if !slices.Contains(validLogLevels, c.LogLevel) {
		errs = append(errs, fmt.Errorf("invalid log level '%s', must be one of: %v", c.LogLevel, validLogLevels))
	}

	if len(errs) > 0 {
		return fmt.Errorf("configuration validation failed: %v", errs)
	}

	return nil
}

// ApplyDefaults applies default values to unset fields
func (c *Config) ApplyDefaults() {
	if c.OpenAIModel == "" {
		c.OpenAIModel = "gpt-4"
	}

	if c.MaxQueueSize == 0 {
		c.MaxQueueSize = 100
	}

	if c.WorkerCount == 0 {
		c.WorkerCount = 2
	}

	if c.RetryAttempts == 0 {
		c.RetryAttempts = 3
	}

	if c.LogLevel == "" {
		c.LogLevel = "info"
	}
}

// LoadFromEnv loads configuration values from environment variables
func (c *Config) LoadFromEnv() error {
	// Load string values
	if val := os.Getenv("HEALER_OPENAI_API_KEY"); val != "" {
		c.OpenAIAPIKey = val
	}
	if val := os.Getenv("HEALER_OPENAI_MODEL"); val != "" {
		c.OpenAIModel = val
	}
	if val := os.Getenv("HEALER_GITHUB_TOKEN"); val != "" {
		c.GitHubToken = val
	}
	if val := os.Getenv("HEALER_REPO_OWNER"); val != "" {
		c.RepoOwner = val
	}
	if val := os.Getenv("HEALER_REPO_NAME"); val != "" {
		c.RepoName = val
	}
	if val := os.Getenv("HEALER_LOG_LEVEL"); val != "" {
		c.LogLevel = val
	}

	// Load boolean values
	if val := os.Getenv("HEALER_ENABLED"); val != "" {
		enabled, err := strconv.ParseBool(val)
		if err != nil {
			return fmt.Errorf("invalid HEALER_ENABLED value '%s': must be true or false", val)
		}
		c.Enabled = enabled
	}

	// Load integer values
	if val := os.Getenv("HEALER_MAX_QUEUE_SIZE"); val != "" {
		size, err := strconv.Atoi(val)
		if err != nil {
			return fmt.Errorf("invalid HEALER_MAX_QUEUE_SIZE value '%s': must be a number", val)
		}
		c.MaxQueueSize = size
	}

	if val := os.Getenv("HEALER_WORKER_COUNT"); val != "" {
		count, err := strconv.Atoi(val)
		if err != nil {
			return fmt.Errorf("invalid HEALER_WORKER_COUNT value '%s': must be a number", val)
		}
		c.WorkerCount = count
	}

	if val := os.Getenv("HEALER_RETRY_ATTEMPTS"); val != "" {
		attempts, err := strconv.Atoi(val)
		if err != nil {
			return fmt.Errorf("invalid HEALER_RETRY_ATTEMPTS value '%s': must be a number", val)
		}
		c.RetryAttempts = attempts
	}

	return nil
}

// ValidateAPIKeys validates API keys and repository settings
func (c *Config) ValidateAPIKeys() error {
	var errs []error

	if c.Enabled {
		// Validate OpenAI API key format (should start with sk-)
		if c.OpenAIAPIKey != "" && !strings.HasPrefix(c.OpenAIAPIKey, "sk-") {
			errs = append(errs, errors.New("OpenAI API key should start with 'sk-'"))
		}

		// Validate GitHub token format (should be non-empty and reasonable length)
		if c.GitHubToken != "" && len(c.GitHubToken) < 10 {
			errs = append(errs, errors.New("GitHub token appears to be too short"))
		}

		// Validate repository settings format
		if c.RepoOwner != "" && (strings.Contains(c.RepoOwner, "/") || strings.Contains(c.RepoOwner, " ")) {
			errs = append(errs, errors.New("repository owner should not contain '/' or spaces"))
		}

		if c.RepoName != "" && (strings.Contains(c.RepoName, "/") || strings.Contains(c.RepoName, " ")) {
			errs = append(errs, errors.New("repository name should not contain '/' or spaces"))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("API key validation failed: %v", errs)
	}

	return nil
}

// LoadConfig loads configuration from JSON file and environment variables
func LoadConfig(configPath string) (*Config, error) {
	config := DefaultConfig()

	// Load from JSON file if provided
	if configPath != "" {
		if err := config.LoadFromFile(configPath); err != nil {
			return nil, fmt.Errorf("failed to load config from file: %w", err)
		}
	}

	// Load from environment variables (overrides file values)
	if err := config.LoadFromEnv(); err != nil {
		return nil, fmt.Errorf("failed to load config from environment: %w", err)
	}

	// Apply defaults for any missing values
	config.ApplyDefaults()

	// Validate the final configuration
	if err := config.ValidateComplete(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return &config, nil
}

// LoadFromFile loads configuration from a JSON file
func (c *Config) LoadFromFile(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("config file not found: %s", filePath)
		}
		return fmt.Errorf("failed to read config file: %w", err)
	}

	if err := json.Unmarshal(data, c); err != nil {
		return fmt.Errorf("failed to parse JSON config: %w", err)
	}

	return nil
}

// ValidateComplete performs comprehensive validation with clear error messages
func (c *Config) ValidateComplete() error {
	var errs []error

	// Basic validation
	if err := c.Validate(); err != nil {
		errs = append(errs, err)
	}

	// API key validation
	if err := c.ValidateAPIKeys(); err != nil {
		errs = append(errs, err)
	}

	// Additional comprehensive validation
	if c.Enabled {
		// Check for required fields with specific error messages
		if c.OpenAIAPIKey == "" {
			errs = append(errs, errors.New("OpenAI API key is required when healer is enabled. Set HEALER_OPENAI_API_KEY environment variable or provide in config file"))
		}

		if c.GitHubToken == "" {
			errs = append(errs, errors.New("GitHub token is required when healer is enabled. Set HEALER_GITHUB_TOKEN environment variable or provide in config file"))
		}

		if c.RepoOwner == "" {
			errs = append(errs, errors.New("repository owner is required when healer is enabled. Set HEALER_REPO_OWNER environment variable or provide in config file"))
		}

		if c.RepoName == "" {
			errs = append(errs, errors.New("repository name is required when healer is enabled. Set HEALER_REPO_NAME environment variable or provide in config file"))
		}
	}

	// Validate ranges with helpful messages
	if c.MaxQueueSize > 10000 {
		errs = append(errs, errors.New("max queue size should not exceed 10000 to prevent excessive memory usage"))
	}

	if c.WorkerCount > 50 {
		errs = append(errs, errors.New("worker count should not exceed 50 to prevent resource exhaustion"))
	}

	if c.RetryAttempts > 10 {
		errs = append(errs, errors.New("retry attempts should not exceed 10 to prevent excessive delays"))
	}

	if len(errs) > 0 {
		var errorMessages []string
		for _, err := range errs {
			errorMessages = append(errorMessages, err.Error())
		}
		return fmt.Errorf("configuration validation failed:\n- %s", strings.Join(errorMessages, "\n- "))
	}

	return nil
}

// GetFallbackConfig returns a minimal configuration that disables features when required settings are missing
func GetFallbackConfig() *Config {
	config := DefaultConfig()

	// Load what we can from environment
	_ = config.LoadFromEnv() // Ignore errors for fallback

	// Apply defaults
	config.ApplyDefaults()

	// Disable if required settings are missing
	if config.OpenAIAPIKey == "" || config.GitHubToken == "" || config.RepoOwner == "" || config.RepoName == "" {
		config.Enabled = false
	}

	return &config
}

// LogConfigStatus logs the current configuration status for debugging
func (c *Config) LogConfigStatus() []string {
	var status []string

	if c.Enabled {
		status = append(status, "Healer is ENABLED")

		if c.OpenAIAPIKey != "" {
			status = append(status, "✓ OpenAI API key configured")
		} else {
			status = append(status, "✗ OpenAI API key missing")
		}

		if c.GitHubToken != "" {
			status = append(status, "✓ GitHub token configured")
		} else {
			status = append(status, "✗ GitHub token missing")
		}

		if c.RepoOwner != "" && c.RepoName != "" {
			status = append(status, fmt.Sprintf("✓ Repository configured: %s/%s", c.RepoOwner, c.RepoName))
		} else {
			status = append(status, "✗ Repository not configured")
		}

		status = append(status, fmt.Sprintf("Queue size: %d, Workers: %d, Retries: %d", c.MaxQueueSize, c.WorkerCount, c.RetryAttempts))
	} else {
		status = append(status, "Healer is DISABLED - will only log panics")
	}

	return status
}
