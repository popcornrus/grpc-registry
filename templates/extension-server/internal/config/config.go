package config

import (
	"flag"
	"os"
	"strconv"
)

// Config represents the application configuration
type Config struct {
	// Server settings
	Port             int
	LogLevel         string
	DataDir          string
	UseTLS           bool
	TLSCertFile      string
	TLSKeyFile       string

	// Extension settings
	ExtensionID      string
	
	// Registration settings
	AutoRegister              bool
	ProxyAddr                 string
	RegistrationAddr          string
	RegistrationTimeoutSeconds int
	RegistrationDelaySeconds   int
	RegistrationMaxRetries     int
	
	// Client settings for proxy communication
	ClientTimeoutSeconds     int
	MaxRetries               int
}

// NewConfig creates a new Config instance with defaults
func NewConfig() *Config {
	return &Config{
		Port:                      53000,
		LogLevel:                  "info",
		DataDir:                   "./data",
		UseTLS:                    false,
		TLSCertFile:               "",
		TLSKeyFile:                "",
		ExtensionID:               "my-extension-server",
		AutoRegister:              true,
		ProxyAddr:                 "localhost:50051",
		RegistrationAddr:          "",
		RegistrationTimeoutSeconds: 30,
		RegistrationDelaySeconds:   5,
		RegistrationMaxRetries:     5,
		ClientTimeoutSeconds:      10,
		MaxRetries:                3,
	}
}

// ParseFlags parses command-line flags and environment variables
func (c *Config) ParseFlags() {
	// Server settings
	port := flag.Int("port", getEnvInt("PORT", c.Port), "Port to listen on")
	logLevel := flag.String("log-level", getEnvString("LOG_LEVEL", c.LogLevel), "Log level (debug, info, warn, error)")
	dataDir := flag.String("data-dir", getEnvString("DATA_DIR", c.DataDir), "Directory to store data")
	useTLS := flag.Bool("use-tls", getEnvBool("USE_TLS", c.UseTLS), "Use TLS for connections")
	tlsCertFile := flag.String("tls-cert", getEnvString("TLS_CERT_FILE", c.TLSCertFile), "TLS certificate file")
	tlsKeyFile := flag.String("tls-key", getEnvString("TLS_KEY_FILE", c.TLSKeyFile), "TLS key file")

	// Extension settings
	extensionID := flag.String("extension-id", getEnvString("EXTENSION_ID", c.ExtensionID), "Extension server ID")

	// Registration settings
	autoRegister := flag.Bool("auto-register", getEnvBool("AUTO_REGISTER", c.AutoRegister), "Auto-register with proxy manager")
	proxyAddr := flag.String("proxy-addr", getEnvString("PROXY_ADDR", c.ProxyAddr), "Proxy manager address")
	registrationAddr := flag.String("registration-addr", getEnvString("REGISTRATION_ADDR", c.RegistrationAddr), "Address to register with (defaults to hostname:port)")
	regTimeout := flag.Int("reg-timeout", getEnvInt("REG_TIMEOUT_SECONDS", c.RegistrationTimeoutSeconds), "Registration timeout in seconds")
	regDelay := flag.Int("reg-delay", getEnvInt("REG_DELAY_SECONDS", c.RegistrationDelaySeconds), "Registration delay in seconds")
	regRetries := flag.Int("reg-retries", getEnvInt("REG_MAX_RETRIES", c.RegistrationMaxRetries), "Maximum registration retry attempts")

	// Client settings
	clientTimeout := flag.Int("client-timeout", getEnvInt("CLIENT_TIMEOUT_SECONDS", c.ClientTimeoutSeconds), "Client timeout in seconds")
	maxRetries := flag.Int("max-retries", getEnvInt("MAX_RETRIES", c.MaxRetries), "Maximum retry attempts")

	flag.Parse()

	// Update config with parsed values
	c.Port = *port
	c.LogLevel = *logLevel
	c.DataDir = *dataDir
	c.UseTLS = *useTLS
	c.TLSCertFile = *tlsCertFile
	c.TLSKeyFile = *tlsKeyFile
	c.ExtensionID = *extensionID
	c.AutoRegister = *autoRegister
	c.ProxyAddr = *proxyAddr
	c.RegistrationAddr = *registrationAddr
	c.RegistrationTimeoutSeconds = *regTimeout
	c.RegistrationDelaySeconds = *regDelay
	c.RegistrationMaxRetries = *regRetries
	c.ClientTimeoutSeconds = *clientTimeout
	c.MaxRetries = *maxRetries
}

// Helper functions to parse environment variables

func getEnvString(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value, exists := os.LookupEnv(key); exists {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}
