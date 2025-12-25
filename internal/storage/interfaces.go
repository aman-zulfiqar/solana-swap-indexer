package storage

import (
	"context"
	"io"

	"github.com/aman-zulfiqar/solana-swap-indexer/internal/models"
)

// SwapCache defines the interface for caching swap data
type SwapCache interface {
	// AddRecentSwap adds a swap to the recent swaps list
	AddRecentSwap(ctx context.Context, swap *models.SwapEvent) error

	// UpdatePrice updates the current price for a token
	UpdatePrice(ctx context.Context, token string, price float64) error

	// GetRecentSwaps retrieves the most recent swaps
	GetRecentSwaps(ctx context.Context, limit int64) ([]*models.SwapEvent, error)

	// GetPrice retrieves the current price for a token
	GetPrice(ctx context.Context, token string) (float64, error)

	// Ping checks if the cache is reachable
	Ping(ctx context.Context) error

	// Close closes the cache connection
	io.Closer

	// PublishSwap publishes a swap event to the Pub/Sub channel
	PublishSwap(ctx context.Context, swap *models.SwapEvent) error

	// SubscribeSwaps subscribes to real-time swap events
	SubscribeSwaps(ctx context.Context) (<-chan *models.SwapEvent, error)
}

// SwapStore defines the interface for persistent swap storage
type SwapStore interface {
	// InsertSwap inserts a swap event into the store
	InsertSwap(ctx context.Context, swap *models.SwapEvent) error

	// Ping checks if the store is reachable
	Ping(ctx context.Context) error

	// Close closes the store connection
	io.Closer
}

// SwapHandler is a function that processes swap events
type SwapHandler func(*models.SwapEvent)

// StreamProvider defines the interface for swap event streaming
type StreamProvider interface {
	// Start begins streaming swap events
	Start(ctx context.Context, handler SwapHandler) error

	// Stop stops the stream provider
	Stop() error
}
