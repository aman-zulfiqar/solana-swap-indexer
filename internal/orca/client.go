package orca

import (
	"context"
	"fmt"

	"github.com/gagliardetto/solana-go"

	"github.com/aman-zulfiqar/solana-swap-indexer/internal/rpc"
)

// Client provides RPC helpers for fetching Orca pool vault balances
type Client struct {
	rpcClient *rpc.Client
}

// NewClient creates an Orca client using the project's RPC client
func NewClient(cfg rpc.ClientConfig) (*Client, error) {
	rpcClient := rpc.NewClient(cfg)

	return &Client{
		rpcClient: rpcClient,
	}, nil
}

// FetchVaultBalances fetches token account balances for pool vaults
// This is the ONLY RPC method you need for legacy pools with static config
func (c *Client) FetchVaultBalances(
	ctx context.Context,
	vaultA, vaultB solana.PublicKey,
) (balanceA, balanceB uint64, err error) {

	balA, err := c.getTokenAccountBalance(ctx, vaultA)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to fetch vault A balance: %w", err)
	}

	balB, err := c.getTokenAccountBalance(ctx, vaultB)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to fetch vault B balance: %w", err)
	}

	return balA, balB, nil
}

// getTokenAccountBalance calls getTokenAccountBalance RPC method
func (c *Client) getTokenAccountBalance(
	ctx context.Context,
	account solana.PublicKey,
) (uint64, error) {

	// Use your project's RPC client.Call signature:
	// Call(ctx, method, params, result) error

	var result struct {
		Context struct {
			Slot uint64 `json:"slot"`
		} `json:"context"`
		Value struct {
			Amount         string   `json:"amount"`
			Decimals       uint8    `json:"decimals"`
			UiAmount       *float64 `json:"uiAmount"`
			UiAmountString string   `json:"uiAmountString"`
		} `json:"value"`
		Error *rpc.RPCError `json:"error"`
	}

	params := []interface{}{account.String()}

	err := c.rpcClient.Call(ctx, "getTokenAccountBalance", params, &result)
	if err != nil {
		return 0, fmt.Errorf("RPC call failed: %w", err)
	}
	if result.Error != nil {
		return 0, fmt.Errorf("getTokenAccountBalance error: %s", result.Error.Message)
	}

	// Parse amount string to uint64
	var amount uint64
	_, err = fmt.Sscanf(result.Value.Amount, "%d", &amount)
	if err != nil {
		return 0, fmt.Errorf("invalid amount format: %w", err)
	}

	return amount, nil
}

// Close cleans up resources (if your RPC client needs cleanup)
func (c *Client) Close() error {
	// Add cleanup if needed
	return nil
}
