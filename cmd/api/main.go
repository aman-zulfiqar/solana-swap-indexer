package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"

	"github.com/aman-zulfiqar/solana-swap-indexer/internal/ai"
	"github.com/aman-zulfiqar/solana-swap-indexer/internal/cache"
	"github.com/aman-zulfiqar/solana-swap-indexer/internal/config"
	"github.com/aman-zulfiqar/solana-swap-indexer/internal/flags"
	"github.com/aman-zulfiqar/solana-swap-indexer/internal/jupiter"
	"github.com/aman-zulfiqar/solana-swap-indexer/internal/server"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

// env bootstrap function
func loadEnv(logger *logrus.Logger) {
	// Get the project root directory (where go.mod is)
	_, filename, _, _ := runtime.Caller(0)
	projectRoot := filepath.Join(filepath.Dir(filename), "../..")
	envPath := filepath.Join(projectRoot, ".env")

	if err := godotenv.Load(envPath); err != nil {
		logger.Warnf("no .env file found at %s, using system environment variables", envPath)
	} else {
		logger.Infof("loaded .env from %s", envPath)
	}
}

// main is the entry point for the API server
// It initializes all dependencies and starts the HTTP server with graceful shutdown
func main() {
	// Initialize structured logger with custom formatting
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})
	logger.SetLevel(logrus.InfoLevel)

	// load .env BEFORE anything reads os.Getenv
	loadEnv(logger)

	// Load and validate configuration from environment variables
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		logger.WithError(err).Fatal("invalid configuration")
	}

	// Extract API-specific configuration
	apiAddr := cfg.APIAddr
	apiKey := cfg.APIKey
	devMode := cfg.DevMode

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling for graceful shutdown (Ctrl+C, SIGTERM)
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	// Initialize Redis client for caching and feature flags
	rclient := redis.NewClient(&redis.Options{
		Addr: cfg.RedisAddr,
		DB:   0, // Use default database for main application
	})
	if err := rclient.Ping(ctx).Err(); err != nil {
		logger.WithError(err).Fatal("failed to connect to Redis")
	}

	// Initialize swap cache for recent swaps and price data
	swapCache := cache.NewRedisCacheFromClient(rclient, logger)

	// Initialize feature flags store for runtime configuration
	flagStore, err := flags.NewStore(rclient)
	if err != nil {
		logger.WithError(err).Fatal("failed to create flags store")
	}

	// Initialize AI agent for natural language queries (optional)
	var agent *ai.Agent
	aiBase := ai.AgentConfig{
		ClickHouseAddr:     cfg.ClickHouseAddr,
		ClickHouseDatabase: cfg.ClickHouseDatabase,
		ClickHouseUsername: cfg.ClickHouseUsername,
		ClickHousePassword: cfg.ClickHousePassword,
		OpenRouterAPIKey:   cfg.OpenRouterAPIKey,
		Model:              "openai/gpt-4.1-mini", // Default model for NL→SQL translation
		Logger:             logger,
	}

	// Only initialize AI if OpenRouter API key is provided
	if cfg.OpenRouterAPIKey != "" {
		a, err := ai.NewAgent(ctx, aiBase)
		if err != nil {
			logger.WithError(err).Warn("failed to initialize ai agent")
		} else {
			agent = a
			defer func() {
				_ = agent.Close() // Clean up AI resources on shutdown
			}()
		}
	}

	// Create handlers with all dependencies injected
	h := &server.Handlers{
		Cache:        swapCache, // Redis-backed swap data cache
		Flags:        flagStore, // Redis-backed feature flags
		AI:           agent,     // Optional AI agent (can be nil)
		AIBaseConfig: aiBase,    // Base AI configuration for model overrides
		DevMode:      devMode,   // Enable detailed error responses in development
		Logger:       logger,    // Structured logger
		Jupiter:      jupiter.NewClient(os.Getenv("JUPITER_BASE_URL"), os.Getenv("JUPITER_API_KEY")),
	}

	// Create HTTP server with configuration and handlers
	srv, err := server.NewServer(server.ServerDeps{
		Handlers: h,
		Config: server.ServerConfig{
			Addr:    apiAddr, // Server bind address (e.g., ":8090")
			DevMode: devMode, // Development mode flag
			APIKey:  apiKey,  // Optional API key for authentication
		},
	})
	if err != nil {
		logger.WithError(err).Fatal("failed to create http server")
	}

	// Setup graceful shutdown in a separate goroutine
	go func() {
		<-sigCh // Wait for shutdown signal
		logger.Info("shutting down")
		cancel()                               // Cancel context to stop ongoing operations
		_ = srv.Shutdown(context.Background()) // Gracefully shutdown HTTP server
	}()

	// Start the HTTP server
	logger.WithField("addr", apiAddr).Info("api server starting")
	if err := srv.Start(); err != nil {
		// "http: Server closed" is expected during graceful shutdown
		if err.Error() == "http: Server closed" {
			return
		}
		logger.WithError(err).Fatal("api server failed")
	}

	// Wait for server to be fully shut down
	if err := srv.WaitClosed(context.Background()); err != nil {
		fmt.Println(err)
	}
}
