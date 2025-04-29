package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds configuration values for the proxy manager
type Config struct {
	// Port for the proxy manager's gRPC server
	Port int
	
	// TLS certificate file path (empty for insecure)
	TLSCertFile string
	
	// TLS key file path (empty for insecure)
	TLSKeyFile string
	
	// Log level (debug, info, warn, error)
	LogLevel string
	
	// Default connection timeout for gRPC clients
	DefaultTimeout time.Duration
	
	// Default max retries for gRPC clients
	DefaultMaxRetries int
}

// New creates a new Config with values from environment variables or defaults
func New() *Config {
	return &Config{
		Port:             getEnvInt("PROXY_PORT", 50051),
		TLSCertFile:      getEnvString("TLS_CERT", ""),
		TLSKeyFile:       getEnvString("TLS_KEY", ""),
		LogLevel:         getEnvString("LOG_LEVEL", "info"),
		DefaultTimeout:   time.Duration(getEnvInt("DEFAULT_TIMEOUT_SECONDS", 10)) * time.Second,
		DefaultMaxRetries: getEnvInt("DEFAULT_MAX_RETRIES", 3),
	}
}

// getEnvString gets a string value from an environment variable or returns the default
func getEnvString(key, defaultValue string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue
	}
	return value
}

// getEnvInt gets an integer value from an environment variable or returns the default
func getEnvInt(key string, defaultValue int) int {
	valueStr := getEnvString(key, "")
	if valueStr == "" {
		return defaultValue
	}
	
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}
	
	return value
}
