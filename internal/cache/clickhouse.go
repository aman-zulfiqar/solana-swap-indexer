package cache

import (
	"context"
	"fmt"

	"github.com/aman-zulfiqar/solana-swap-indexer/internal/models"
	"github.com/sirupsen/logrus"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// ClickHouseStore implements the SwapStore interface using ClickHouse
type ClickHouseStore struct {
	conn   driver.Conn
	logger *logrus.Logger
}

// ClickHouseConfig holds configuration for ClickHouse connection
type ClickHouseConfig struct {
	Addr     string
	Database string
	Username string
	Password string
	Logger   *logrus.Logger
}

// NewClickHouseStore creates a new ClickHouse store with connection verification
func NewClickHouseStore(ctx context.Context, cfg ClickHouseConfig) (*ClickHouseStore, error) {
	if cfg.Logger == nil {
		cfg.Logger = logrus.New()
	}

	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{cfg.Addr},
		Auth: clickhouse.Auth{
			Database: cfg.Database,
			Username: cfg.Username,
			Password: cfg.Password,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to ClickHouse: %w", err)
	}

	// Verify connection
	if err := conn.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping ClickHouse: %w", err)
	}

	cfg.Logger.WithFields(logrus.Fields{
		"addr":     cfg.Addr,
		"database": cfg.Database,
	}).Info("connected to ClickHouse")

	return &ClickHouseStore{
		conn:   conn,
		logger: cfg.Logger,
	}, nil
}

// InsertSwap inserts a swap event into ClickHouse
func (c *ClickHouseStore) InsertSwap(ctx context.Context, swap *models.SwapEvent) error {
	query := `
		INSERT INTO swaps (
			signature, timestamp, pair, token_in, token_out,
			amount_in, amount_out, price, fee, pool, dex
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	err := c.conn.Exec(ctx, query,
		swap.Signature,
		swap.Timestamp,
		swap.Pair,
		swap.TokenIn,
		swap.TokenOut,
		swap.AmountIn,
		swap.AmountOut,
		swap.Price,
		swap.Fee,
		swap.Pool,
		swap.Dex,
	)

	if err != nil {
		return fmt.Errorf("failed to insert swap: %w", err)
	}

	c.logger.WithFields(logrus.Fields{
		"signature": swap.Signature[:8],
		"pair":      swap.Pair,
	}).Debug("inserted swap into ClickHouse")

	return nil
}

// Ping checks if ClickHouse is reachable
func (c *ClickHouseStore) Ping(ctx context.Context) error {
	return c.conn.Ping(ctx)
}

// Close closes the ClickHouse connection
func (c *ClickHouseStore) Close() error {
	c.logger.Debug("closing ClickHouse connection")
	return c.conn.Close()
}
