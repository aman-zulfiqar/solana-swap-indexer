package swapengine

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/gagliardetto/solana-go"
)

// RiskConfig defines risk management parameters
type RiskConfig struct {
	// Per-transaction limits
	MaxSwapAmountSOL float64 // Max SOL value per swap

	// Daily limits (rolling 24h window)
	DailyLimitSOL float64 // Max SOL value per day

	// Price impact limits
	MaxPriceImpactBps uint16 // Max price impact in bps (e.g., 500 = 5%)

	// Slippage constraints
	DefaultSlippageBps uint16 // Default slippage (e.g., 100 = 1%)
	MaxSlippageBps     uint16 // Max allowed slippage (e.g., 1000 = 10%)

	// Token whitelist (empty = allow all)
	AllowedTokens []string

	// Safety features
	RequireSimulation bool    // Always simulate before sending
	MinBalanceSOL     float64 // Min wallet balance to keep
}

// DefaultRiskConfig returns conservative risk settings
func DefaultRiskConfig() RiskConfig {
	return RiskConfig{
		MaxSwapAmountSOL:   1.0,  // 1 SOL per transaction
		DailyLimitSOL:      10.0, // 10 SOL per day
		MaxPriceImpactBps:  500,  // 5% max price impact
		DefaultSlippageBps: 100,  // 1% default slippage
		MaxSlippageBps:     1000, // 10% max slippage
		AllowedTokens:      []string{"SOL", "USDC", "USDT"},
		RequireSimulation:  true,
		MinBalanceSOL:      0.05, // Keep 0.05 SOL for fees
	}
}

func (rm *RiskManager) getTokenSymbol(mint solana.PublicKey) string {
	m := mint.String()
	for sym, mintStr := range TokenMints {
		if mintStr == m {
			return sym
		}
	}
	// fallback: keep it deterministic for logs/debug; also ensures whitelist fails for unknowns
	return m
}

// RiskManager enforces risk limits
type RiskManager struct {
	config       RiskConfig
	dailyTracker *DailyLimitTracker
}

// NewRiskManager creates a risk manager with the given config
func NewRiskManager(config RiskConfig) *RiskManager {
	return &RiskManager{
		config:       config,
		dailyTracker: NewDailyLimitTracker(),
	}
}

// CheckSwap validates a swap against all risk rules
func (rm *RiskManager) CheckSwap(
	ctx context.Context,
	params *SwapParams,
	quote *QuoteResult,
	walletBalanceSOL float64,
) (*RiskCheckResult, error) {

	result := &RiskCheckResult{
		Allowed:           true,
		MaxSwapAmountSOL:  rm.config.MaxSwapAmountSOL,
		DailyLimitSOL:     rm.config.DailyLimitSOL,
		MaxPriceImpactBps: rm.config.MaxPriceImpactBps,
		WhitelistedTokens: rm.config.AllowedTokens,
	}

	// 1. Check per-transaction limit
	swapValueSOL := rm.estimateSwapValueSOL(params, quote)
	if swapValueSOL > rm.config.MaxSwapAmountSOL {
		result.Allowed = false
		result.ExceedsMaxSwapAmount = true
		result.Reason = fmt.Sprintf("swap value %.4f SOL exceeds max %.4f SOL per transaction",
			swapValueSOL, rm.config.MaxSwapAmountSOL)
		return result, nil
	}

	// 2. Check daily limit
	dailyUsed := rm.dailyTracker.GetDailyUsage()
	result.DailyUsedSOL = dailyUsed
	result.DailyRemainingSOL = rm.config.DailyLimitSOL - dailyUsed

	if dailyUsed+swapValueSOL > rm.config.DailyLimitSOL {
		result.Allowed = false
		result.ExceedsDailyLimit = true
		result.Reason = fmt.Sprintf("daily limit exceeded: used %.4f + %.4f > %.4f SOL",
			dailyUsed, swapValueSOL, rm.config.DailyLimitSOL)
		return result, nil
	}

	// 3. Check token whitelist
	if len(rm.config.AllowedTokens) > 0 {
		inputSymbol := rm.getTokenSymbol(params.InputMint)
		outputSymbol := rm.getTokenSymbol(params.OutputMint)

		if !rm.isTokenAllowed(inputSymbol) || !rm.isTokenAllowed(outputSymbol) {
			result.Allowed = false
			result.TokenNotWhitelisted = true
			result.Reason = fmt.Sprintf("token not whitelisted: %s or %s",
				inputSymbol, outputSymbol)
			return result, nil
		}
	}

	// 4. Check price impact
	if quote.PriceImpact*10000 > float64(rm.config.MaxPriceImpactBps) {
		result.Allowed = false
		result.PriceImpactTooHigh = true
		result.ActualPriceImpact = quote.PriceImpact
		result.Reason = fmt.Sprintf("price impact %.2f%% exceeds max %.2f%%",
			quote.PriceImpact*100, float64(rm.config.MaxPriceImpactBps)/100)
		return result, nil
	}

	// 5. Check minimum balance (ensure enough for fees)
	if walletBalanceSOL-swapValueSOL < rm.config.MinBalanceSOL {
		result.Allowed = false
		result.Reason = fmt.Sprintf("insufficient balance: would leave %.4f SOL, need %.4f SOL minimum",
			walletBalanceSOL-swapValueSOL, rm.config.MinBalanceSOL)
		return result, nil
	}

	// 6. Validate slippage
	if params.SlippageBps > rm.config.MaxSlippageBps {
		result.Allowed = false
		result.Reason = fmt.Sprintf("slippage %d bps exceeds max %d bps",
			params.SlippageBps, rm.config.MaxSlippageBps)
		return result, nil
	}

	return result, nil
}

// RecordSwap records a successful swap for daily limit tracking
func (rm *RiskManager) RecordSwap(params *SwapParams, quote *QuoteResult) {
	swapValueSOL := rm.estimateSwapValueSOL(params, quote)
	rm.dailyTracker.RecordSwap(swapValueSOL)
}

// estimateSwapValueSOL converts swap amount to SOL equivalent
func (rm *RiskManager) estimateSwapValueSOL(params *SwapParams, quote *QuoteResult) float64 {
	// If input is SOL, use that directly
	if params.InputMint.String() == TokenMints["SOL"] {
		decimals := TokenDecimals["SOL"]
		denom := math.Pow10(int(decimals))
		return float64(params.AmountIn) / denom
	}

	// If output is SOL, use that
	if params.OutputMint.String() == TokenMints["SOL"] {
		decimals := TokenDecimals["SOL"]
		denom := math.Pow10(int(decimals))
		return float64(quote.AmountOut) / denom
	}

	// MVP fallback: treat non-SOL swaps as small constant SOL value
	return 0.01
}

// isTokenAllowed checks if a token is in the whitelist
func (rm *RiskManager) isTokenAllowed(symbol string) bool {
	if len(rm.config.AllowedTokens) == 0 {
		return true // No whitelist = allow all
	}

	for _, allowed := range rm.config.AllowedTokens {
		if allowed == symbol {
			return true
		}
	}
	return false
}

// DailyLimitTracker tracks rolling 24-hour usage
type DailyLimitTracker struct {
	swaps []swapRecord
}

type swapRecord struct {
	timestamp time.Time
	amountSOL float64
}

// NewDailyLimitTracker creates a new tracker
func NewDailyLimitTracker() *DailyLimitTracker {
	return &DailyLimitTracker{
		swaps: make([]swapRecord, 0),
	}
}

// RecordSwap adds a swap to the tracker
func (t *DailyLimitTracker) RecordSwap(amountSOL float64) {
	t.swaps = append(t.swaps, swapRecord{
		timestamp: time.Now(),
		amountSOL: amountSOL,
	})

	// Clean up old records
	t.cleanup()
}

// GetDailyUsage calculates total usage in the last 24 hours
func (t *DailyLimitTracker) GetDailyUsage() float64 {
	t.cleanup()

	total := 0.0
	for _, swap := range t.swaps {
		total += swap.amountSOL
	}
	return total
}

// cleanup removes swaps older than 24 hours
func (t *DailyLimitTracker) cleanup() {
	cutoff := time.Now().Add(-24 * time.Hour)

	newSwaps := make([]swapRecord, 0, len(t.swaps))
	for _, swap := range t.swaps {
		if swap.timestamp.After(cutoff) {
			newSwaps = append(newSwaps, swap)
		}
	}

	t.swaps = newSwaps
}

// GetSwapHistory returns recent swaps
func (t *DailyLimitTracker) GetSwapHistory() []swapRecord {
	t.cleanup()
	return t.swaps
}

// Reset clears all tracked swaps (for testing)
func (t *DailyLimitTracker) Reset() {
	t.swaps = make([]swapRecord, 0)
}
