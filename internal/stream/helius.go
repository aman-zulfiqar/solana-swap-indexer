// ============================================================================
// stream/helius.go - Helius WebSocket Client (FREE TIER)
// ============================================================================
package stream

import (
	"context"
	"fmt"
	"log"
	"time"

	"solana-swap-indexer/internal/models"

	"github.com/gorilla/websocket"
)

type HeliusStream struct {
	apiKey  string
	conn    *websocket.Conn
	handler func(*models.SwapEvent)
}

func NewHeliusStream(apiKey string) *HeliusStream {
	return &HeliusStream{
		apiKey: apiKey,
	}
}

// Connect to Helius WebSocket
func (h *HeliusStream) Connect(ctx context.Context) error {
	url := fmt.Sprintf("wss://atlas-mainnet.helius-rpc.com/?api-key=%s", h.apiKey)

	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return fmt.Errorf("websocket dial: %w", err)
	}

	h.conn = conn

	// Subscribe to transaction mentions for popular DEX programs
	subscribeMsg := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "transactionSubscribe",
		"params": []interface{}{
			map[string]interface{}{
				"accountInclude": []string{
					"675kPX9MHTjS2zt1qfr1NYHuzeLXfQM9H24wFSUt1Mp8", // Raydium AMM
					"9W959DqEETiGZocYWCQPaJ6sBmUzgfxXfqGeTEdp3aQP", // Orca Whirlpool
				},
			},
			map[string]interface{}{
				"commitment":                     "confirmed",
				"encoding":                       "jsonParsed",
				"transactionDetails":             "full",
				"showRewards":                    false,
				"maxSupportedTransactionVersion": 0,
			},
		},
	}

	if err := conn.WriteJSON(subscribeMsg); err != nil {
		return fmt.Errorf("subscribe: %w", err)
	}

	log.Println("✅ Connected to Helius WebSocket")
	return nil
}

// Start listening for transactions
func (h *HeliusStream) Listen(ctx context.Context, handler func(*models.SwapEvent)) error {
	h.handler = handler

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			var msg map[string]interface{}
			if err := h.conn.ReadJSON(&msg); err != nil {
				log.Printf("Read error: %v", err)
				time.Sleep(5 * time.Second)
				continue
			}

			// Parse transaction and extract swap data
			if swap := h.parseTransaction(msg); swap != nil {
				handler(swap)
			}
		}
	}
}

// Parse transaction into SwapEvent
func (h *HeliusStream) parseTransaction(data map[string]interface{}) *models.SwapEvent {
	// This is simplified - you'll need to parse based on your DEX instruction format
	// For now, return a mock swap event

	params, ok := data["params"].(map[string]interface{})
	if !ok {
		return nil
	}

	result, ok := params["result"].(map[string]interface{})
	if !ok {
		return nil
	}

	value, ok := result["value"].(map[string]interface{})
	if !ok {
		return nil
	}

	// Extract signature
	signature, _ := value["signature"].(string)

	// Mock parsing - replace with actual instruction parsing
	return &models.SwapEvent{
		Signature: signature,
		Timestamp: time.Now(),
		Pair:      "SOL/USDC",
		TokenIn:   "SOL",
		TokenOut:  "USDC",
		AmountIn:  1.5,
		AmountOut: 297.75,
		Price:     198.50,
		Fee:       0.0025,
		Pool:      "RaydiumAMM",
		Dex:       "Raydium",
	}
}
