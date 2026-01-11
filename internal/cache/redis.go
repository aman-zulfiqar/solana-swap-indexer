package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/aman-zulfiqar/solana-swap-indexer/internal/constants"
	"github.com/aman-zulfiqar/solana-swap-indexer/internal/models"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

// RedisCache implements the SwapCache interface using Redis
type RedisCache struct {
	client *redis.Client
	logger *logrus.Logger
}

// RedisConfig holds configuration for Redis connection
type RedisConfig struct {
	Addr   string
	Logger *logrus.Logger
}

// NewRedisCache creates a new Redis cache with connection verification
func NewRedisCache(ctx context.Context, cfg RedisConfig) (*RedisCache, error) {
	if cfg.Logger == nil {
		cfg.Logger = logrus.New()
	}

	client := redis.NewClient(&redis.Options{
		Addr: cfg.Addr,
		DB:   0,
	})

	// Verify connection
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	cfg.Logger.WithField("addr", cfg.Addr).Info("connected to Redis")
	return NewRedisCacheFromClient(client, cfg.Logger), nil
}
func NewRedisCacheFromClient(client *redis.Client, logger *logrus.Logger) *RedisCache {
	if logger == nil {
		logger = logrus.New()
	}
	return &RedisCache{
		client: client,
		logger: logger,
	}
}

// AddRecentSwap adds a swap to the recent swaps list
func (r *RedisCache) AddRecentSwap(ctx context.Context, swap *models.SwapEvent) error {
	data, err := json.Marshal(swap)
	if err != nil {
		return fmt.Errorf("failed to marshal swap: %w", err)
	}

	// Add to list (LPUSH = add to front)
	if err := r.client.LPush(ctx, constants.RedisKeyRecentSwaps, data).Err(); err != nil {
		return fmt.Errorf("failed to push to Redis: %w", err)
	}

	// Trim to keep only last N swaps
	if err := r.client.LTrim(ctx, constants.RedisKeyRecentSwaps, 0, int64(constants.MaxRecentSwaps-1)).Err(); err != nil {
		return fmt.Errorf("failed to trim list: %w", err)
	}

	r.logger.WithFields(logrus.Fields{
		"signature": swap.Signature[:8],
		"pair":      swap.Pair,
	}).Debug("added swap to cache")

	return nil
}

// UpdatePrice updates the current price for a token
func (r *RedisCache) UpdatePrice(ctx context.Context, token string, price float64) error {
	key := constants.RedisKeyPricePrefix + token

	if err := r.client.Set(ctx, key, price, 0).Err(); err != nil {
		return fmt.Errorf("failed to set price: %w", err)
	}

	r.logger.WithFields(logrus.Fields{
		"token": token,
		"price": price,
	}).Debug("updated token price")

	return nil
}

// GetRecentSwaps retrieves the most recent swaps
func (r *RedisCache) GetRecentSwaps(ctx context.Context, limit int64) ([]*models.SwapEvent, error) {
	data, err := r.client.LRange(ctx, constants.RedisKeyRecentSwaps, 0, limit-1).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get recent swaps: %w", err)
	}

	swaps := make([]*models.SwapEvent, 0, len(data))
	for _, d := range data {
		var swap models.SwapEvent
		if err := json.Unmarshal([]byte(d), &swap); err != nil {
			r.logger.WithError(err).Warn("failed to unmarshal swap from cache")
			continue
		}
		swaps = append(swaps, &swap)
	}

	return swaps, nil
}

// GetPrice retrieves the current price for a token
func (r *RedisCache) GetPrice(ctx context.Context, token string) (float64, error) {
	key := constants.RedisKeyPricePrefix + token

	val, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("failed to get price: %w", err)
	}

	price, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse price: %w", err)
	}

	return price, nil
}

// Ping checks if Redis is reachable
func (r *RedisCache) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

// Close closes the Redis connection
func (r *RedisCache) Close() error {
	r.logger.Debug("closing Redis connection")
	return r.client.Close()
}

// PublishSwap publishes a swap event to the Pub/Sub channel for real-time consumers
func (r *RedisCache) PublishSwap(ctx context.Context, swap *models.SwapEvent) error {
	data, err := json.Marshal(swap)
	if err != nil {
		return fmt.Errorf("failed to marshal swap for publish: %w", err)
	}

	subscribers, err := r.client.Publish(ctx, constants.PubSubChannelSwaps, data).Result()
	if err != nil {
		return fmt.Errorf("failed to publish swap: %w", err)
	}

	r.logger.WithFields(logrus.Fields{
		"signature":   swap.Signature[:8],
		"pair":        swap.Pair,
		"subscribers": subscribers,
	}).Debug("published swap to channel")

	return nil
}

// SubscribeSwaps creates a subscription to the swaps channel and returns a channel
// that receives swap events in real-time. The caller is responsible for reading
// from the channel until the context is cancelled.
func (r *RedisCache) SubscribeSwaps(ctx context.Context) (<-chan *models.SwapEvent, error) {
	pubsub := r.client.Subscribe(ctx, constants.PubSubChannelSwaps)

	// Verify subscription is active
	_, err := pubsub.Receive(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to swaps channel: %w", err)
	}

	r.logger.WithField("channel", constants.PubSubChannelSwaps).Info("subscribed to swaps channel")

	// Create buffered output channel
	swapChan := make(chan *models.SwapEvent, 100)

	// Start goroutine to read messages and forward to output channel
	go func() {
		defer close(swapChan)
		defer func() {
			if err := pubsub.Close(); err != nil {
				r.logger.WithError(err).Warn("error closing pubsub subscription")
			}
		}()

		ch := pubsub.Channel()
		for {
			select {
			case <-ctx.Done():
				r.logger.Debug("subscription context cancelled, closing")
				return

			case msg, ok := <-ch:
				if !ok {
					r.logger.Warn("pubsub channel closed unexpectedly")
					return
				}

				var swap models.SwapEvent
				if err := json.Unmarshal([]byte(msg.Payload), &swap); err != nil {
					r.logger.WithError(err).Warn("failed to unmarshal swap from pubsub")
					continue
				}

				// Non-blocking send to avoid blocking the pubsub reader
				select {
				case swapChan <- &swap:
				default:
					r.logger.Warn("swap channel buffer full, dropping message")
				}
			}
		}
	}()

	return swapChan, nil
}
