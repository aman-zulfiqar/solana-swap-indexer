// ============================================================================
// models/swap.go
// ============================================================================
package models

import "time"

type SwapEvent struct {
	Signature string    `json:"signature"`
	Timestamp time.Time `json:"timestamp"`
	Pair      string    `json:"pair"`
	TokenIn   string    `json:"token_in"`
	TokenOut  string    `json:"token_out"`
	AmountIn  float64   `json:"amount_in"`
	AmountOut float64   `json:"amount_out"`
	Price     float64   `json:"price"`
	Fee       float64   `json:"fee"`
	Pool      string    `json:"pool"`
	Dex       string    `json:"dex"` // e.g., "Raydium", "Orca"
}
