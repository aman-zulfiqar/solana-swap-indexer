package cache

import (
	"context"
	"fmt"
	"log"

	"solana-swap-indexer/internal/models"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

type ClickHouseStore struct {
	conn driver.Conn
}

func NewClickHouseStore(addr string) (*ClickHouseStore, error) {
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{addr},
		Auth: clickhouse.Auth{
			Database: "solana",
			Username: "default",
			Password: "",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to ClickHouse: %w", err)
	}

	// Test connection
	if err := conn.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping ClickHouse: %w", err)
	}

	log.Println("✅ Connected to ClickHouse")

	return &ClickHouseStore{
		conn: conn,
	}, nil
}

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

	return nil
}
