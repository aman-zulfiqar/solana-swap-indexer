package orca

import (
	"github.com/gagliardetto/solana-go"
)

// Legacy Orca constant-product pool program ID
const (
	LegacyProgramID = "9W959DqEETiGZocYWCQPaJ6sBmUzgfxXfqGeTEdp3aQP"
)

// SwapQuote contains quote details for a swap
type SwapQuote struct {
	PoolName     string
	InputMint    solana.PublicKey
	OutputMint   solana.PublicKey
	AmountIn     uint64  // Raw input amount (with decimals)
	AmountOut    uint64  // Expected output (with decimals)
	MinAmountOut uint64  // Minimum output after slippage
	FeeBps       uint16  // Fee in basis points
	PriceImpact  float64 // Price impact percentage (0.01 = 1%)
	ReserveIn    uint64  // Input reserve before swap
	ReserveOut   uint64  // Output reserve before swap
}

// PoolState represents current on-chain state (reserves)
type PoolState struct {
	Pool      *LegacyPool
	ReserveA  uint64 // Current balance in vault A
	ReserveB  uint64 // Current balance in vault B
	Timestamp int64  // When fetched
}
