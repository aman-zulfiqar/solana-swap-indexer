// ============================================================================
// cmd/subscriber/main.go - Example Subscriber (Consumer)
// ============================================================================
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"solana-swap-indexer/internal/cache"
	"solana-swap-indexer/internal/models"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	pubsub := cache.NewPubSubManager("localhost:6379")

	log.Println("👂 Starting swap subscriber...")

	// Subscribe to all swaps
	go pubsub.Subscribe(ctx, "swaps:all", func(swap *models.SwapEvent) {
		log.Printf("📨 Received: %s | %s | %.2f %s -> %.2f %s | Price: %.2f",
			swap.Signature[:8], swap.Pair, swap.AmountIn, swap.TokenIn,
			swap.AmountOut, swap.TokenOut, swap.Price)
	})

	// Subscribe to specific pair
	go pubsub.Subscribe(ctx, "swaps:pair:SOL/USDC", func(swap *models.SwapEvent) {
		log.Printf("💰 SOL/USDC Swap: %.2f @ %.2f", swap.AmountIn, swap.Price)
	})

	// Subscribe to pattern (all pairs)
	go pubsub.PSubscribe(ctx, "swaps:pair:*", func(swap *models.SwapEvent) {
		log.Printf("🔍 Pattern match: %s", swap.Pair)
	})

	log.Println("✅ Subscriber running. Press Ctrl+C to stop.")

	<-sigChan
	log.Println("🛑 Shutting down subscriber...")
}
