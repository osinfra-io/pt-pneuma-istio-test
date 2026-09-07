package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all configuration for the istio-test application
type Config struct {
	// Server configuration
	Server ServerConfig

	// Metadata service configuration
	Metadata MetadataConfig

	// Observability configuration
	Observability ObservabilityConfig
}

// ServerConfig holds HTTP server related configuration
type ServerConfig struct {
	Port         string        `json:"port"`
	ReadTimeout  time.Duration `json:"read_timeout"`
	WriteTimeout time.Duration `json:"write_timeout"`
	IdleTimeout  time.Duration `json:"idle_timeout"`
}

// MetadataConfig holds metadata service related configuration
type MetadataConfig struct {
	HTTPTimeout     time.Duration `json:"http_timeout"`
	MaxRetries      int           `json:"max_retries"`
	BaseRetryDelay  time.Duration `json:"base_retry_delay"`
	MaxRetryDelay   time.Duration `json:"max_retry_delay"`
	RetryMultiplier float64       `json:"retry_multiplier"`
}

// ObservabilityConfig holds observability related configuration
type ObservabilityConfig struct {
	LogLevel           string        `json:"log_level"`
	EnableProfiler     bool          `json:"enable_profiler"`
	EnableTracing      bool          `json:"enable_tracing"`
	EnablePIIRedaction bool          `json:"enable_pii_redaction"`
	ShutdownTimeout    time.Duration `json:"shutdown_timeout"`
}

// Validate validates the entire configuration
func (c *Config) Validate() error {
	if err := validateServerConfig(c.Server); err != nil {
		return err
	}
	if err := validateMetadataConfig(c.Metadata); err != nil {
		return err
	}
	if err := validateObservabilityConfig(c.Observability); err != nil {
		return err
	}
	return nil
}

// Load creates a new Config instance with values from environment variables
// and sensible defaults
func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Port:         getEnv("PORT", "8080"),
			ReadTimeout:  getDuration("SERVER_READ_TIMEOUT", 5*time.Second),
			WriteTimeout: getDuration("SERVER_WRITE_TIMEOUT", 10*time.Second),
			IdleTimeout:  getDuration("SERVER_IDLE_TIMEOUT", 60*time.Second),
		},
		Metadata: MetadataConfig{
			HTTPTimeout:     getDuration("METADATA_HTTP_TIMEOUT", 10*time.Second),
			MaxRetries:      getInt("METADATA_MAX_RETRIES", 3),
			BaseRetryDelay:  getDuration("METADATA_BASE_RETRY_DELAY", 100*time.Millisecond),
			MaxRetryDelay:   getDuration("METADATA_MAX_RETRY_DELAY", 2*time.Second),
			RetryMultiplier: getFloat("METADATA_RETRY_MULTIPLIER", 2.0),
		},
		Observability: ObservabilityConfig{
			LogLevel:           getEnv("LOG_LEVEL", "info"),
			EnableProfiler:     getBool("ENABLE_PROFILER", true),
			EnableTracing:      getBool("ENABLE_TRACING", true),
			EnablePIIRedaction: getBool("ENABLE_PII_REDACTION", true),
			ShutdownTimeout:    getDuration("SHUTDOWN_TIMEOUT", 5*time.Second),
		},
	}
}

// getEnv returns the value of an environment variable or a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getDuration parses a duration from an environment variable or returns a default value
func getDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			if duration > 0 {
				return duration
			}
		}
	}
	return defaultValue
}

// getInt parses an integer from an environment variable or returns a default value
func getInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			if intValue >= 0 {
				return intValue
			}
		}
	}
	return defaultValue
}

// getFloat parses a float from an environment variable or returns a default value
func getFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			if floatValue > 0 {
				return floatValue
			}
		}
	}
	return defaultValue
}

// getBool parses a boolean from an environment variable or returns a default value
func getBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

// validateServerConfig validates ServerConfig fields
func validateServerConfig(sc ServerConfig) error {
	// Validate port is a valid port number
	if port, err := strconv.Atoi(sc.Port); err != nil {
		return fmt.Errorf("invalid server port '%s': must be a number", sc.Port)
	} else if port < 1 || port > 65535 {
		return fmt.Errorf("invalid server port %d: must be between 1 and 65535", port)
	}

	// Validate timeouts are positive
	if sc.ReadTimeout <= 0 {
		return fmt.Errorf("invalid server read timeout: must be positive")
	}
	if sc.WriteTimeout <= 0 {
		return fmt.Errorf("invalid server write timeout: must be positive")
	}
	if sc.IdleTimeout <= 0 {
		return fmt.Errorf("invalid server idle timeout: must be positive")
	}

	return nil
}

// validateMetadataConfig validates MetadataConfig fields
func validateMetadataConfig(mc MetadataConfig) error {
	// Validate HTTP timeout is positive
	if mc.HTTPTimeout <= 0 {
		return fmt.Errorf("invalid metadata HTTP timeout: must be positive")
	}

	// Validate max retries is non-negative
	if mc.MaxRetries < 0 {
		return fmt.Errorf("invalid metadata max retries: must be non-negative")
	}

	// Validate retry delays are positive
	if mc.BaseRetryDelay <= 0 {
		return fmt.Errorf("invalid metadata base retry delay: must be positive")
	}
	if mc.MaxRetryDelay <= 0 {
		return fmt.Errorf("invalid metadata max retry delay: must be positive")
	}

	// Validate base delay is not greater than max delay
	if mc.BaseRetryDelay > mc.MaxRetryDelay {
		return fmt.Errorf("invalid metadata retry delays: base delay (%v) cannot be greater than max delay (%v)", mc.BaseRetryDelay, mc.MaxRetryDelay)
	}

	// Validate retry multiplier is greater than 1
	if mc.RetryMultiplier <= 1.0 {
		return fmt.Errorf("invalid metadata retry multiplier: must be greater than 1.0")
	}

	return nil
}

// validateObservabilityConfig validates ObservabilityConfig fields
func validateObservabilityConfig(oc ObservabilityConfig) error {
	// Validate log level is one of the standard levels
	validLogLevels := []string{"trace", "debug", "info", "warn", "warning", "error", "fatal", "panic"}
	found := false
	for _, level := range validLogLevels {
		if strings.ToLower(oc.LogLevel) == level {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("invalid log level '%s': must be one of %s", oc.LogLevel, strings.Join(validLogLevels, ", "))
	}
	// Validate shutdown timeout is positive
	if oc.ShutdownTimeout <= 0 {
		return fmt.Errorf("invalid shutdown timeout: must be positive")
	}

	return nil
}
