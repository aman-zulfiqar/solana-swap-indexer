// ============================================================================
// cache/pubsub.go - Redis Pub/Sub Wrapper
// ============================================================================
package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"solana-swap-indexer/internal/models"

	"github.com/redis/go-redis/v9"
)

type PubSubManager struct {
	client *redis.Client
}

func NewPubSubManager(addr string) *PubSubManager {
	return &PubSubManager{
		client: redis.NewClient(&redis.Options{
			Addr: addr,
			DB:   0,
		}),
	}
}

// Publish swap event to multiple channels
func (p *PubSubManager) PublishSwap(ctx context.Context, swap *models.SwapEvent) error {
	data, err := json.Marshal(swap)
	if err != nil {
		return err
	}

	// Publish to multiple channels for different subscribers
	channels := []string{
		"swaps:all",                             // All swaps
		fmt.Sprintf("swaps:pair:%s", swap.Pair), // Pair-specific
		fmt.Sprintf("swaps:dex:%s", swap.Dex),   // DEX-specific
		"price:updates",                         // Price feed
	}

	pipe := p.client.Pipeline()
	for _, channel := range channels {
		pipe.Publish(ctx, channel, data)
	}

	_, err = pipe.Exec(ctx)
	return err
}

// Subscribe to a channel
func (p *PubSubManager) Subscribe(ctx context.Context, channel string, handler func(*models.SwapEvent)) error {
	pubsub := p.client.Subscribe(ctx, channel)
	defer pubsub.Close()

	log.Printf("📡 Subscribed to channel: %s", channel)

	ch := pubsub.Channel()
	for msg := range ch {
		var swap models.SwapEvent
		if err := json.Unmarshal([]byte(msg.Payload), &swap); err != nil {
			log.Printf("Error unmarshaling swap: %v", err)
			continue
		}

		handler(&swap)
	}

	return nil
}

// Subscribe to pattern (e.g., "swaps:pair:*")
func (p *PubSubManager) PSubscribe(ctx context.Context, pattern string, handler func(*models.SwapEvent)) error {
	pubsub := p.client.PSubscribe(ctx, pattern)
	defer pubsub.Close()

	log.Printf("📡 Subscribed to pattern: %s", pattern)

	ch := pubsub.Channel()
	for msg := range ch {
		var swap models.SwapEvent
		if err := json.Unmarshal([]byte(msg.Payload), &swap); err != nil {
			log.Printf("Error unmarshaling swap: %v", err)
			continue
		}

		handler(&swap)
	}

	return nil
}
