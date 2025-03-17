package main

import (
	"os"
	"strconv"
	"time"
)

// Config holds the DuckDB server configuration
type Config struct {
	DatabasePath   string
	MaxConnections int
	QueryTimeout   time.Duration
	AuthToken      string
	LogLevel       string
}

// LoadConfig loads configuration from environment variables
func LoadConfig() *Config {
	config := &Config{
		DatabasePath:   getEnv("DUCKDB_PATH", ":memory:"),
		MaxConnections: getEnvAsInt("MAX_CONNECTIONS", 10),
		QueryTimeout:   time.Duration(getEnvAsInt("QUERY_TIMEOUT", 30)) * time.Second,
		AuthToken:      getEnv("AUTH_TOKEN", ""),
		LogLevel:       getEnv("LOG_LEVEL", "info"),
	}
	return config
}

// GetDatabasePath returns the configured database path
func (c *Config) GetDatabasePath() string {
	return c.DatabasePath
}

// GetMaxConnections returns the maximum number of allowed connections
func (c *Config) GetMaxConnections() int {
	return c.MaxConnections
}

// GetQueryTimeout returns the configured query timeout duration
func (c *Config) GetQueryTimeout() time.Duration {
	return c.QueryTimeout
}

// GetAuthToken returns the configured authentication token
func (c *Config) GetAuthToken() string {
	return c.AuthToken
}

// Helper function to get environment variable with default value
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// Helper function to get environment variable as integer with default value
func getEnvAsInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
