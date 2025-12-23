// ============================================================================
// cmd/indexer/main.go - Main Indexer Service
// ============================================================================
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"solana-swap-indexer/internal/cache"
	"solana-swap-indexer/internal/models"
	"solana-swap-indexer/internal/stream"
)

type Indexer struct {
	redis      *cache.RedisCache
	clickhouse *cache.ClickHouseStore
	pubsub     *cache.PubSubManager
}

func NewIndexer() (*Indexer, error) {
	redis := cache.NewRedisCache("localhost:6379")

	clickhouse, err := cache.NewClickHouseStore("localhost:9000")
	if err != nil {
		return nil, err
	}

	pubsub := cache.NewPubSubManager("localhost:6379")

	return &Indexer{
		redis:      redis,
		clickhouse: clickhouse,
		pubsub:     pubsub,
	}, nil
}

func (idx *Indexer) ProcessSwap(ctx context.Context, swap *models.SwapEvent) error {
	log.Printf("📊 Processing swap: %s - %s (%.2f %s -> %.2f %s)",
		swap.Signature[:8], swap.Pair, swap.AmountIn, swap.TokenIn,
		swap.AmountOut, swap.TokenOut)

	// 1. Store in Redis cache
	if err := idx.redis.AddRecentSwap(ctx, swap); err != nil {
		log.Printf("⚠️  Redis cache error: %v", err)
	}

	// 2. Update price feed
	if err := idx.redis.UpdatePrice(ctx, swap.TokenOut, swap.Price); err != nil {
		log.Printf("⚠️  Price update error: %v", err)
	}

	// 3. Publish to Redis Pub/Sub (real-time distribution)
	if err := idx.pubsub.PublishSwap(ctx, swap); err != nil {
		log.Printf("⚠️  Pub/Sub error: %v", err)
	}

	// 4. Store in ClickHouse (historical data)
	if err := idx.clickhouse.InsertSwap(ctx, swap); err != nil {
		log.Printf("❌ ClickHouse error: %v", err)
		return err
	}

	log.Printf("✅ Swap processed successfully")
	return nil
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Initialize indexer
	indexer, err := NewIndexer()
	if err != nil {
		log.Fatal(err)
	}

	log.Println("🚀 Starting Solana Swap Indexer...")

	// Get stream provider from env
	streamProvider := os.Getenv("STREAM_PROVIDER") // "helius", "rpc", or "triton"
	if streamProvider == "" {
		streamProvider = "rpc" // default to free RPC
	}

	switch streamProvider {
	case "helius":
		apiKey := os.Getenv("HELIUS_API_KEY")
		if apiKey == "" {
			log.Fatal("HELIUS_API_KEY required when using helius provider")
		}
		log.Printf("📡 Using Helius WebSocket (API Key: %s...)", apiKey[:8])
		helius := stream.NewHeliusStream(apiKey)
		if err := helius.Connect(ctx); err != nil {
			log.Fatal(err)
		}
		go helius.Listen(ctx, func(swap *models.SwapEvent) {
			indexer.ProcessSwap(ctx, swap)
		})

	case "triton":
		apiKey := os.Getenv("TRITON_API_KEY")
		rpcURL := fmt.Sprintf("https://api.mainnet.solana.triton.one/%s", apiKey)
		if apiKey == "" {
			log.Fatal("TRITON_API_KEY required when using triton provider")
		}
		log.Printf("📡 Using Triton RPC Polling")
		poller := stream.NewRPCPoller(rpcURL)
		go poller.Poll(ctx, func(swap *models.SwapEvent) {
			indexer.ProcessSwap(ctx, swap)
		})

	case "rpc":
		rpcURL := os.Getenv("SOLANA_RPC_URL")
		if rpcURL == "" {
			rpcURL = "https://api.mainnet-beta.solana.com"
		}
		log.Printf("📡 Using Public RPC Polling: %s", rpcURL)
		poller := stream.NewRPCPoller(rpcURL)
		go poller.Poll(ctx, func(swap *models.SwapEvent) {
			indexer.ProcessSwap(ctx, swap)
		})

	default:
		log.Fatalf("Unknown stream provider: %s", streamProvider)
	}

	log.Println("✅ Indexer running. Press Ctrl+C to stop.")

	// Wait for shutdown signal
	<-sigChan
	log.Println("🛑 Shutting down gracefully...")
	cancel()
}
