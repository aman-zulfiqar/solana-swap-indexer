package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	// RPC settings
	RPCUrl       string
	PollInterval time.Duration

	// Redis settings
	RedisAddr string

	// ClickHouse settings
	ClickHouseAddr     string
	ClickHouseDatabase string
	ClickHouseUsername string
	ClickHousePassword string

	// HTTP client settings
	HTTPTimeout  time.Duration
	MaxRetries   int
	RetryBackoff time.Duration

	// Stream provider
	StreamProvider string
	TritonAPIKey   string

	// LLM / OpenRouter settings
	OpenRouterAPIKey string

	// API
	APIAddr string
	APIKey  string
	DevMode bool
}

// Load reads all configuration from environment variables
// Validates all required vars first, then panics with complete list if any are missing
func Load() *Config {
	// Validate all required env vars first
	validateRequiredEnvVars()

	return &Config{
		// RPC
		RPCUrl:       mustEnv("SOLANA_RPC_URL"),
		PollInterval: mustDurationEnv("POLL_INTERVAL"),

		// Redis
		RedisAddr: mustEnv("REDIS_ADDR"),

		// ClickHouse
		ClickHouseAddr:     mustEnv("CLICKHOUSE_ADDR"),
		ClickHouseDatabase: mustEnv("CLICKHOUSE_DATABASE"),
		ClickHouseUsername: mustEnv("CLICKHOUSE_USERNAME"),
		ClickHousePassword: mustEnv("CLICKHOUSE_PASSWORD"),

		// HTTP
		HTTPTimeout:  mustDurationEnv("HTTP_TIMEOUT"),
		MaxRetries:   mustIntEnv("MAX_RETRIES"),
		RetryBackoff: mustDurationEnv("RETRY_BACKOFF"),

		// Stream
		StreamProvider: mustEnv("STREAM_PROVIDER"),
		TritonAPIKey:   mustEnv("TRITON_API_KEY"),

		// LLM / OpenRouter
		OpenRouterAPIKey: mustEnv("OPENROUTER_API_KEY"),

		// API
		APIAddr: mustEnv("API_ADDR"),
		APIKey:  mustEnv("API_KEY"),
		DevMode: mustBoolEnv("DEV"),
	}
}

// validateRequiredEnvVars checks all required env vars and panics with complete list if any are missing
func validateRequiredEnvVars() {
	required := []string{
		"SOLANA_RPC_URL",
		"POLL_INTERVAL",
		"REDIS_ADDR",
		"CLICKHOUSE_ADDR",
		"CLICKHOUSE_DATABASE",
		"CLICKHOUSE_USERNAME",
		"CLICKHOUSE_PASSWORD",
		"HTTP_TIMEOUT",
		"MAX_RETRIES",
		"RETRY_BACKOFF",
		"STREAM_PROVIDER",
		"TRITON_API_KEY",
		"OPENROUTER_API_KEY",
		"API_ADDR",
		"API_KEY",
		"DEV",
	}

	var missing []string
	for _, key := range required {
		val := strings.TrimSpace(os.Getenv(key))
		if val == "" {
			missing = append(missing, key)
		}
	}

	if len(missing) > 0 {
		panic(fmt.Sprintf(
			"missing required environment variables:\n  %s\n\nPlease set all required variables in your .env file.",
			strings.Join(missing, "\n  "),
		))
	}
}

// mustEnv reads a required string env or panics
func mustEnv(key string) string {
	val := strings.TrimSpace(os.Getenv(key))
	if val == "" {
		panic(fmt.Sprintf("missing required environment variable: %s", key))
	}
	return val
}

// mustIntEnv reads a required int env or panics
func mustIntEnv(key string) int {
	val := mustEnv(key)
	intVal, err := strconv.Atoi(val)
	if err != nil {
		panic(fmt.Sprintf("invalid integer for %s: %v (got: %q)", key, err, val))
	}
	return intVal
}

// mustDurationEnv reads a required duration env or panics
func mustDurationEnv(key string) time.Duration {
	val := mustEnv(key)
	durationVal, err := time.ParseDuration(val)
	if err != nil {
		panic(fmt.Sprintf("invalid duration for %s: %v (got: %q). Examples: 30s, 5m, 1h", key, err, val))
	}
	return durationVal
}

// mustBoolEnv reads a required bool env or panics
func mustBoolEnv(key string) bool {
	val := mustEnv(key)
	boolVal, err := strconv.ParseBool(val)
	if err != nil {
		panic(fmt.Sprintf("invalid boolean for %s: %v (got: %q). Must be: true, false, 1, 0, t, f", key, err, val))
	}
	return boolVal
}

// Validate is optional since all fields are mustEnv-driven
func (c *Config) Validate() error {
	return nil
}
