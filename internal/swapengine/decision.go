package swapengine

import (
	"fmt"
	"math"
	"time"

	"github.com/gagliardetto/solana-go"
)

type DecisionEngine struct {
	risk RiskConfig
}

func NewDecisionEngine(risk RiskConfig) *DecisionEngine {
	return &DecisionEngine{risk: risk}
}

func (de *DecisionEngine) ValidateIntent(intent *SwapIntent) error {
	if intent == nil {
		return fmt.Errorf("intent is nil")
	}
	if intent.InputToken == "" || intent.OutputToken == "" {
		return fmt.Errorf("input/output token required")
	}
	if intent.InputToken == intent.OutputToken {
		return fmt.Errorf("input and output token must differ")
	}
	if intent.Amount <= 0 {
		return fmt.Errorf("amount must be > 0")
	}
	if _, ok := TokenMints[intent.InputToken]; !ok {
		return fmt.Errorf("unknown input token: %s", intent.InputToken)
	}
	if _, ok := TokenMints[intent.OutputToken]; !ok {
		return fmt.Errorf("unknown output token: %s", intent.OutputToken)
	}
	return nil
}

func (de *DecisionEngine) EnrichIntent(intent *SwapIntent) {
	if intent.RequestedAt.IsZero() {
		intent.RequestedAt = time.Now()
	}
	if intent.SlippageBps == nil {
		v := de.risk.DefaultSlippageBps
		intent.SlippageBps = &v
	}
	if intent.MaxPriceImpactBps == nil {
		v := de.risk.MaxPriceImpactBps
		intent.MaxPriceImpactBps = &v
	}
}

func (de *DecisionEngine) ParseIntent(intent *SwapIntent) (*SwapParams, error) {
	if err := de.ValidateIntent(intent); err != nil {
		return nil, err
	}
	de.EnrichIntent(intent)

	inMint := solana.MustPublicKeyFromBase58(TokenMints[intent.InputToken])
	outMint := solana.MustPublicKeyFromBase58(TokenMints[intent.OutputToken])

	inDecimals := TokenDecimals[intent.InputToken]
	amountIn := toRawAmount(intent.Amount, inDecimals)

	params := &SwapParams{
		InputMint:         inMint,
		OutputMint:        outMint,
		AmountIn:          amountIn,
		MinAmountOut:      0,  // executor fills after quoting + slippage
		PoolName:          "", // executor selects by mints unless caller sets
		SlippageBps:       *intent.SlippageBps,
		MaxPriceImpactBps: *intent.MaxPriceImpactBps,
		Intent:            intent,
		ParsedAt:          time.Now(),
		ValidUntil:        time.Now().Add(2 * time.Minute),
	}
	return params, nil
}

func toRawAmount(amount float64, decimals uint8) uint64 {
	if amount <= 0 {
		return 0
	}
	mul := math.Pow10(int(decimals))
	return uint64(math.Round(amount * mul))
}
