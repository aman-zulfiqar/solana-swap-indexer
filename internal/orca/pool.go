package orca

import (
	"context"
	"time"
)

// RefreshPoolState fetches current vault balances for a pool
func RefreshPoolState(
	ctx context.Context,
	client *Client,
	pool *LegacyPool,
) (*PoolState, error) {

	reserveA, reserveB, err := client.FetchVaultBalances(ctx, pool.VaultA, pool.VaultB)
	if err != nil {
		return nil, err
	}

	return &PoolState{
		Pool:      pool,
		ReserveA:  reserveA,
		ReserveB:  reserveB,
		Timestamp: time.Now().Unix(),
	}, nil
}

// GetReserves returns reserves in the correct order for a swap direction
func (ps *PoolState) GetReserves(aToB bool) (reserveIn, reserveOut uint64) {
	if aToB {
		return ps.ReserveA, ps.ReserveB
	}
	return ps.ReserveB, ps.ReserveA
}
