package orca

import (
	"fmt"
	"math"
	"math/big"
)

// CalculateLegacySwapOutput computes output for constant-product AMM (legacy pools)
// Uses x * y = k formula with fees applied to input
// Returns (amountOut, priceImpact, error)
func CalculateLegacySwapOutput(
	amountIn uint64,
	reserveIn uint64,
	reserveOut uint64,
	feeNumerator uint64,
	feeDenominator uint64,
) (uint64, float64, error) {

	if amountIn == 0 || reserveIn == 0 || reserveOut == 0 {
		return 0, 0, fmt.Errorf("invalid inputs: amounts must be > 0")
	}

	if feeDenominator == 0 {
		return 0, 0, fmt.Errorf("feeDenominator cannot be 0")
	}

	// Apply fee: amountInAfterFee = amountIn * (feeDenominator - feeNumerator) / feeDenominator
	// Use big.Int to prevent overflow
	amountInBig := new(big.Int).SetUint64(amountIn)
	feeMultiplier := new(big.Int).SetUint64(feeDenominator - feeNumerator)
	feeDenom := new(big.Int).SetUint64(feeDenominator)

	amountInAfterFee := new(big.Int).Mul(amountInBig, feeMultiplier)
	amountInAfterFee.Div(amountInAfterFee, feeDenom)

	// Constant product formula: out = (amountInAfterFee * reserveOut) / (reserveIn + amountInAfterFee)
	reserveOutBig := new(big.Int).SetUint64(reserveOut)
	reserveInBig := new(big.Int).SetUint64(reserveIn)

	numerator := new(big.Int).Mul(amountInAfterFee, reserveOutBig)
	denominator := new(big.Int).Add(reserveInBig, amountInAfterFee)

	amountOutBig := new(big.Int).Div(numerator, denominator)

	// Convert back to uint64
	if !amountOutBig.IsUint64() {
		return 0, 0, fmt.Errorf("output amount overflow")
	}
	amountOut := amountOutBig.Uint64()

	// Price impact calculation
	// idealRate = reserveOut / reserveIn
	// executionRate = amountOut / amountIn
	// priceImpact = 1 - (executionRate / idealRate)
	idealRate := float64(reserveOut) / float64(reserveIn)
	executionRate := float64(amountOut) / float64(amountIn)
	priceImpact := 0.0

	if idealRate > 0 {
		priceImpact = math.Max(0, 1-(executionRate/idealRate))
	}

	return amountOut, priceImpact, nil
}

// ApplySlippage calculates minimum output with slippage tolerance
// slippageBps: basis points (e.g., 100 = 1%, 50 = 0.5%)
func ApplySlippage(amountOut uint64, slippageBps uint16) uint64 {
	if slippageBps >= 10000 {
		return 0 // 100% slippage = no output
	}

	// minOut = amountOut * (10000 - slippageBps) / 10000
	slippageFactor := 10000 - uint64(slippageBps)

	amountBig := new(big.Int).SetUint64(amountOut)
	factor := new(big.Int).SetUint64(slippageFactor)
	denom := new(big.Int).SetUint64(10000)

	result := new(big.Int).Mul(amountBig, factor)
	result.Div(result, denom)

	return result.Uint64()
}

// ValidatePriceImpact checks if price impact exceeds threshold
func ValidatePriceImpact(priceImpact float64, maxImpactBps uint16) error {
	maxImpact := float64(maxImpactBps) / 10000.0

	if priceImpact > maxImpact {
		return fmt.Errorf("price impact %.4f%% exceeds max %.4f%%",
			priceImpact*100, maxImpact*100)
	}

	return nil
}

// CalculateFeeBps converts fee numerator/denominator to basis points
func CalculateFeeBps(feeNumerator, feeDenominator uint64) uint16 {
	if feeDenominator == 0 {
		return 0
	}
	return uint16((feeNumerator * 10000) / feeDenominator)
}
