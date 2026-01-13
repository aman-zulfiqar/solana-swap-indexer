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

func main() {
	// Initialize logger
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "15:04:05",
	})

	// load .env BEFORE anything reads os.Getenv
	loadEnv(logger)

	// Set log level from env (default: warn to keep output clean)
	logLevel := os.Getenv("LOG_LEVEL")
	switch logLevel {
	case "debug":
		logger.SetLevel(logrus.DebugLevel)
	case "info":
		logger.SetLevel(logrus.InfoLevel)
	case "error":
		logger.SetLevel(logrus.ErrorLevel)
	default:
		logger.SetLevel(logrus.WarnLevel)
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

	// Connect to Redis
	redisCache, err := cache.NewRedisCache(ctx, cache.RedisConfig{
		Addr:   cfg.RedisAddr,
		Logger: logger,
	})
	if err != nil {
		logger.WithError(err).Fatal("failed to connect to Redis")
	}
	defer redisCache.Close()

	// Subscribe to swaps channel
	swapChan, err := redisCache.SubscribeSwaps(ctx)
	if err != nil {
		logger.WithError(err).Fatal("failed to subscribe to swaps")
	}

	// Print header
	printHeader()

	// Process swaps in background
	go func() {
		for swap := range swapChan {
			printSwap(swap)
		}
	}()

	// Wait for shutdown signal
	<-sigChan
	fmt.Println("\n\nShutting down...")
	cancel()
}

func printHeader() {
	fmt.Println()
	fmt.Println("╔════════════════════════════════════════════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                              Live Swap Viewer - Solana Swap Indexer (Pub/Sub)                              ║")
	fmt.Println("╠════════════════════════════════════════════════════════════════════════════════════════════════════════════╣")
	fmt.Println("║  Time     │ Pair                 │ Amount In              │ Amount Out             │ Price        │ Sig    ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════════════════════════════════════════════════╝")
}

func printSwap(swap *models.SwapEvent) {
	// Truncate pair for display
	pair := swap.Pair
	if len(pair) > 18 {
		pair = pair[:18]
	}

	// Format amounts with token symbols
	amountIn := fmt.Sprintf("%.4f %s", swap.AmountIn, truncateToken(swap.TokenIn))
	amountOut := fmt.Sprintf("%.4f %s", swap.AmountOut, truncateToken(swap.TokenOut))

	// Truncate signature
	sig := swap.Signature
	if len(sig) > 8 {
		sig = sig[:8]
	}

	fmt.Printf("[%s] %-18s │ %20s │ %20s │ %12.6f │ %s\n",
		swap.Timestamp.Format("15:04:05"),
		pair,
		amountIn,
		amountOut,
		swap.Price,
		sig,
	)
}

func truncateToken(token string) string {
	if len(token) > 12 {
		return token[:4] + "..." + token[len(token)-4:]
	}
	return token
}
