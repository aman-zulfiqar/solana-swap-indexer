package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"

	"github.com/aman-zulfiqar/solana-swap-indexer/internal/cache"
	"github.com/aman-zulfiqar/solana-swap-indexer/internal/config"
	"github.com/aman-zulfiqar/solana-swap-indexer/internal/models"
	"github.com/aman-zulfiqar/solana-swap-indexer/internal/rpc"
	"github.com/aman-zulfiqar/solana-swap-indexer/internal/storage"
	"github.com/aman-zulfiqar/solana-swap-indexer/internal/stream"

	"github.com/joho/godotenv"
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

// Indexer orchestrates swap event processing
type Indexer struct {
	cache  storage.SwapCache
	store  storage.SwapStore
	logger *logrus.Logger
}

// NewIndexer creates a new indexer with the given dependencies
func NewIndexer(cache storage.SwapCache, store storage.SwapStore, logger *logrus.Logger) *Indexer {
	return &Indexer{
		cache:  cache,
		store:  store,
		logger: logger,
	}
}

// ProcessSwap handles a single swap event
func (idx *Indexer) ProcessSwap(ctx context.Context, swap *models.SwapEvent) error {
	log := idx.logger.WithFields(logrus.Fields{
		"signature": swap.Signature[:8],
		"pair":      swap.Pair,
		"amount_in": swap.AmountIn,
		"token_in":  swap.TokenIn,
	})

	// Store in cache
	if err := idx.cache.AddRecentSwap(ctx, swap); err != nil {
		log.WithError(err).Warn("failed to cache swap")
	}

	// Update price
	if err := idx.cache.UpdatePrice(ctx, swap.TokenOut, swap.Price); err != nil {
		log.WithError(err).Warn("failed to update price")
	}

	// Store in database
	if err := idx.store.InsertSwap(ctx, swap); err != nil {
		log.WithError(err).Error("failed to store swap")
		return err
	}

	// Publish to Pub/Sub for real-time consumers (non-blocking)
	if err := idx.cache.PublishSwap(ctx, swap); err != nil {
		log.WithError(err).Warn("failed to publish swap to pubsub")
		// Don't return error - publishing is not critical to core functionality
	}

	log.Info("swap processed successfully")
	return nil
}

// Close closes all connections
func (idx *Indexer) Close() error {
	var errs []error

	if err := idx.cache.Close(); err != nil {
		errs = append(errs, fmt.Errorf("cache close: %w", err))
	}

	if err := idx.store.Close(); err != nil {
		errs = append(errs, fmt.Errorf("store close: %w", err))
	}

	if len(errs) > 0 {
		return fmt.Errorf("close errors: %v", errs)
	}

	return nil
}

func main() {
	// Initialize logger
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})

	// load .env BEFORE anything reads os.Getenv
	loadEnv(logger)

	// Set log level from env (default: info)
	logLevel := os.Getenv("LOG_LEVEL")
	switch logLevel {
	case "debug":
		logger.SetLevel(logrus.DebugLevel)
	case "warn":
		logger.SetLevel(logrus.WarnLevel)
	case "error":
		logger.SetLevel(logrus.ErrorLevel)
	default:
		logger.SetLevel(logrus.InfoLevel)
	}

	// Load configuration
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		logger.WithError(err).Fatal("invalid configuration")
	}

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Initialize Redis cache
	redisCache, err := cache.NewRedisCache(ctx, cache.RedisConfig{
		Addr:   cfg.RedisAddr,
		Logger: logger,
	})
	if err != nil {
		logger.WithError(err).Fatal("failed to connect to Redis")
	}

	// Initialize ClickHouse store
	clickhouseStore, err := cache.NewClickHouseStore(ctx, cache.ClickHouseConfig{
		Addr:     cfg.ClickHouseAddr,
		Database: cfg.ClickHouseDatabase,
		Username: cfg.ClickHouseUsername,
		Password: cfg.ClickHousePassword,
		Logger:   logger,
	})
	if err != nil {
		logger.WithError(err).Fatal("failed to connect to ClickHouse")
	}

	// Create indexer
	indexer := NewIndexer(redisCache, clickhouseStore, logger)
	defer func() {
		logger.Info("closing connections")
		if err := indexer.Close(); err != nil {
			logger.WithError(err).Error("error closing connections")
		}
	}()

	// Determine RPC URL based on provider
	rpcURL := cfg.RPCUrl
	if cfg.StreamProvider == "triton" {
		if cfg.TritonAPIKey == "" {
			logger.Fatal("TRITON_API_KEY required when using triton provider")
		}
		rpcURL = fmt.Sprintf("https://api.mainnet.solana.triton.one/%s", cfg.TritonAPIKey)
	}

	// Create RPC client
	rpcClient := rpc.NewClient(rpc.ClientConfig{
		BaseURL:      rpcURL,
		Timeout:      cfg.HTTPTimeout,
		MaxRetries:   cfg.MaxRetries,
		RetryBackoff: cfg.RetryBackoff,
		Logger:       logger,
	})

	// Create poller
	poller := stream.NewRPCPoller(stream.RPCPollerConfig{
		RPCClient:    rpcClient,
		PollInterval: cfg.PollInterval,
		Logger:       logger,
	})

	logger.WithFields(logrus.Fields{
		"provider": cfg.StreamProvider,
		"rpc_url":  rpcURL,
		"interval": cfg.PollInterval,
	}).Info("starting Solana swap indexer")

	// Start polling in background
	go func() {
		if err := poller.Start(ctx, func(swap *models.SwapEvent) {
			if err := indexer.ProcessSwap(ctx, swap); err != nil {
				logger.WithError(err).Error("failed to process swap")
			}
		}); err != nil && err != context.Canceled {
			logger.WithError(err).Error("poller stopped with error")
		}
	}()

	logger.Info("indexer running, press Ctrl+C to stop")

	// Wait for shutdown signal
	<-sigChan
	logger.Info("shutting down gracefully")
	cancel()
}
