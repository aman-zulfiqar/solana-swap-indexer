// ============================================================================
// stream/rpc_poller.go - Free RPC Polling Alternative
// ============================================================================
package stream

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"solana-swap-indexer/internal/models"
)

type RPCPoller struct {
	rpcURL           string
	programAddresses []string
	lastSignature    string
	pollInterval     time.Duration
}

func NewRPCPoller(rpcURL string) *RPCPoller {
	return &RPCPoller{
		rpcURL: rpcURL,
		programAddresses: []string{
			"675kPX9MHTjS2zt1qfr1NYHuzeLXfQM9H24wFSUt1Mp8", // Raydium
		},
		pollInterval: 10 * time.Second, // Slower to avoid rate limits
	}
}

// Start polling for new signatures
func (r *RPCPoller) Poll(ctx context.Context, handler func(*models.SwapEvent)) error {
	ticker := time.NewTicker(r.pollInterval)
	defer ticker.Stop()

	log.Println("🔄 Starting RPC polling...")

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := r.fetchNewTransactions(ctx, handler); err != nil {
				log.Printf("Poll error: %v", err)
			}
		}
	}
}

func (r *RPCPoller) fetchNewTransactions(ctx context.Context, handler func(*models.SwapEvent)) error {
	// Build request params
	requestParams := map[string]interface{}{
		"limit": 10,
	}

	// If we have a lastSignature, only fetch signatures AFTER it (not before)
	// "until" means: get signatures up until this one (i.e., newer signatures)
	if r.lastSignature != "" {
		requestParams["until"] = r.lastSignature
		log.Printf("🔄 Fetching signatures after: %s...", r.lastSignature[:8])
	}

	params := []interface{}{
		r.programAddresses[0],
		requestParams,
	}

	body := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "getSignaturesForAddress",
		"params":  params,
	}

	data, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, "POST", r.rpcURL, bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Parse response
	var rpcResp struct {
		Result []struct {
			Signature string      `json:"signature"`
			Slot      int64       `json:"slot"`
			Err       interface{} `json:"err"`
			BlockTime int64       `json:"blockTime"`
		} `json:"result"`
		Error *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		log.Printf("❌ Failed to decode RPC response: %v", err)
		return err
	}

	if rpcResp.Error != nil {
		log.Printf("❌ RPC error: %s (code: %d)", rpcResp.Error.Message, rpcResp.Error.Code)
		return fmt.Errorf("RPC error: %s", rpcResp.Error.Message)
	}

	if len(rpcResp.Result) == 0 {
		log.Println("✅ No new transactions")
		return nil
	}

	log.Printf("📥 Found %d new signatures", len(rpcResp.Result))

	// Update lastSignature to the most recent one (first in list)
	r.lastSignature = rpcResp.Result[0].Signature

	// Process each signature (for now, mock the swap event)
	for i, sig := range rpcResp.Result {
		if sig.Err != nil {
			log.Printf("⏭️  Skipping failed tx: %s", sig.Signature[:8])
			continue
		}

		log.Printf("🔍 Processing tx %d/%d: %s...", i+1, len(rpcResp.Result), sig.Signature[:8])

		// Mock swap event - in production, you'd call getTransaction to get full details
		swap := &models.SwapEvent{
			Signature: sig.Signature,
			Timestamp: time.Unix(sig.BlockTime, 0),
			Pair:      "SOL/USDC",
			TokenIn:   "SOL",
			TokenOut:  "USDC",
			AmountIn:  2.0,
			AmountOut: 397.00,
			Price:     198.50,
			Fee:       0.0025,
			Pool:      "RaydiumAMM",
			Dex:       "Raydium",
		}

		handler(swap)
	}

	return nil
}
