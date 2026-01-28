package swapengine

import (
	"time"

	"github.com/gagliardetto/solana-go"
)

// SwapIntent represents the AI agent's trading intention
type SwapIntent struct {
	// Core swap parameters
	InputToken  string  // Token symbol (e.g., "SOL", "USDC")
	OutputToken string  // Token symbol
	Amount      float64 // Amount in human-readable units (e.g., 1.5 SOL)

	// Optional parameters (AI can specify or use defaults)
	SlippageBps       *uint16 // Slippage tolerance in basis points (e.g., 100 = 1%)
	MaxPriceImpactBps *uint16 // Max acceptable price impact (e.g., 300 = 3%)

	// Context
	Reason      string    // AI reasoning for the swap
	Confidence  float64   // AI confidence score (0-1)
	RequestedAt time.Time // When intent was generated
}

// SwapParams represents validated, executable swap parameters
type SwapParams struct {
	// Token info
	InputMint  solana.PublicKey
	OutputMint solana.PublicKey

	// Amounts (in raw token units with decimals)
	AmountIn     uint64
	MinAmountOut uint64 // With slippage applied

	// Pool selection
	PoolName string

	// Risk parameters
	SlippageBps       uint16
	MaxPriceImpactBps uint16

	// Metadata
	Intent     *SwapIntent
	ParsedAt   time.Time
	ValidUntil time.Time // Blockhash expiration
}

// SwapExecution represents the complete execution lifecycle
type SwapExecution struct {
	// Identifiers
	ExecutionID string // Unique execution ID
	Signature   string // Transaction signature

	// Parameters
	Params *SwapParams

	// Quote details
	Quote *QuoteResult

	// Execution timeline
	StartedAt   time.Time
	SimulatedAt *time.Time
	SignedAt    *time.Time
	SentAt      *time.Time
	ConfirmedAt *time.Time
	CompletedAt *time.Time

	// Results
	Success      bool
	Error        string
	SimulationOK bool

	// Blockchain details
	Slot         uint64
	BlockTime    *int64
	ComputeUnits uint64
	PriorityFee  uint64

	// Actual amounts (from transaction logs)
	ActualAmountIn  *uint64
	ActualAmountOut *uint64

	// Metadata
	Logs []string
}

// QuoteResult contains detailed quote information
type QuoteResult struct {
	PoolName      string
	AmountIn      uint64
	AmountOut     uint64
	MinAmountOut  uint64
	PriceImpact   float64
	FeeBps        uint16
	ReserveIn     uint64
	ReserveOut    uint64
	ExecutionRate float64 // Output per input
	QuotedAt      time.Time
}

// SwapResult is the final result returned to the caller
type SwapResult struct {
	ExecutionID string
	Signature   string
	Success     bool
	Error       string

	// Quote vs actual
	ExpectedOut uint64
	ActualOut   *uint64

	// Performance metrics
	Duration       time.Duration
	SimulationMS   int64
	ConfirmationMS int64

	// Details
	Quote     *QuoteResult
	Execution *SwapExecution
}

// RiskCheckResult contains risk validation outcome
type RiskCheckResult struct {
	Allowed bool
	Reason  string

	// Per-transaction limits
	ExceedsMaxSwapAmount bool
	MaxSwapAmountSOL     float64

	// Daily limits
	ExceedsDailyLimit bool
	DailyLimitSOL     float64
	DailyUsedSOL      float64
	DailyRemainingSOL float64

	// Token whitelist
	TokenNotWhitelisted bool
	WhitelistedTokens   []string

	// Price impact
	PriceImpactTooHigh bool
	MaxPriceImpactBps  uint16
	ActualPriceImpact  float64
}

// TokenDecimals maps token symbols to their decimal places
var TokenDecimals = map[string]uint8{
	"SOL":  9,
	"USDC": 6,
	"USDT": 6,
	"RAY":  6,
	"SRM":  6,
}

// TokenMints maps token symbols to their mint addresses
var TokenMints = map[string]string{
	"SOL":  "So11111111111111111111111111111111111111112",
	"USDC": "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v",
	"USDT": "Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB",
	// Add more as needed
}
