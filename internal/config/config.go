package config

import (
	"os"
	"strconv"
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
}

func Load() *Config {
	return &Config{
		// RPC
		RPCUrl:       getEnv("SOLANA_RPC_URL", "https://api.mainnet-beta.solana.com"),
		PollInterval: getDurationEnv("POLL_INTERVAL", 30*time.Second),

		// Redis
		RedisAddr: getEnv("REDIS_ADDR", "localhost:6379"),

		// ClickHouse
		ClickHouseAddr:     getEnv("CLICKHOUSE_ADDR", "localhost:9000"),
		ClickHouseDatabase: getEnv("CLICKHOUSE_DATABASE", "solana"),
		ClickHouseUsername: getEnv("CLICKHOUSE_USERNAME", "default"),
		ClickHousePassword: getEnv("CLICKHOUSE_PASSWORD", ""),

		// HTTP
		HTTPTimeout:  getDurationEnv("HTTP_TIMEOUT", 30*time.Second),
		MaxRetries:   getIntEnv("MAX_RETRIES", 5),
		RetryBackoff: getDurationEnv("RETRY_BACKOFF", 2*time.Second),

		// Stream
		StreamProvider: getEnv("STREAM_PROVIDER", "rpc"),
		TritonAPIKey:   getEnv("TRITON_API_KEY", ""),
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getIntEnv(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultVal
}

func getDurationEnv(key string, defaultVal time.Duration) time.Duration {
	if val := os.Getenv(key); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			return d
		}
	}
	return defaultVal
}
