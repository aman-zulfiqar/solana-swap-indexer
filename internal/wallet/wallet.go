package wallet

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	projectrpc "github.com/aman-zulfiqar/solana-swap-indexer/internal/rpc"
	"github.com/gagliardetto/solana-go"
)

// SendOptions configures transaction sending behavior
type SendOptions struct {
	SkipPreflight       bool
	PreflightCommitment string
	MaxRetries          *int
	Commitment          string
}

// DefaultSendOptions returns recommended send settings
func DefaultSendOptions() SendOptions {
	maxRetries := 3
	return SendOptions{
		SkipPreflight:       false,
		PreflightCommitment: "processed",
		MaxRetries:          &maxRetries,
		Commitment:          "confirmed",
	}
}

// SignTx signs a transaction with the wallet's private key
func (w *Wallet) SignTx(tx *solana.Transaction) error {
	_, err := tx.Sign(func(key solana.PublicKey) *solana.PrivateKey {
		if key.Equals(w.pub) {
			return &w.priv
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to sign transaction: %w", err)
	}
	return nil
}

// SendTx sends a signed transaction with configurable options
func (w *Wallet) SendTx(ctx context.Context, tx *solana.Transaction, opts *SendOptions) (string, error) {
	if opts == nil {
		defaultOpts := DefaultSendOptions()
		opts = &defaultOpts
	}

	// Serialize transaction
	txBytes, err := tx.MarshalBinary()
	if err != nil {
		return "", fmt.Errorf("failed to serialize transaction: %w", err)
	}

	// Encode to base64
	encodedTx := base64.StdEncoding.EncodeToString(txBytes)

	// Build RPC params
	params := []any{
		encodedTx,
		map[string]any{
			"encoding":            "base64",
			"skipPreflight":       opts.SkipPreflight,
			"preflightCommitment": opts.PreflightCommitment,
		},
	}

	if opts.MaxRetries != nil {
		params[1].(map[string]any)["maxRetries"] = *opts.MaxRetries
	}

	// Call sendTransaction
	var resp struct {
		Result string               `json:"result"`
		Error  *projectrpc.RPCError `json:"error"`
	}

	if err := w.rpc.Call(ctx, "sendTransaction", params, &resp); err != nil {
		return "", fmt.Errorf("sendTransaction RPC failed: %w", err)
	}

	if resp.Error != nil {
		return "", fmt.Errorf("sendTransaction error: code=%d, message=%s",
			resp.Error.Code, resp.Error.Message)
	}

	return resp.Result, nil
}

// GetLatestBlockhash fetches the most recent blockhash with commitment level
func (w *Wallet) GetLatestBlockhash(ctx context.Context, commitment ...string) (solana.Hash, error) {
	commitmentLevel := "processed"
	if len(commitment) > 0 {
		commitmentLevel = commitment[0]
	}

	var resp struct {
		Result struct {
			Value struct {
				Blockhash            string `json:"blockhash"`
				LastValidBlockHeight uint64 `json:"lastValidBlockHeight"`
			} `json:"value"`
		} `json:"result"`
		Error *projectrpc.RPCError `json:"error"`
	}

	params := []any{
		map[string]any{"commitment": commitmentLevel},
	}

	if err := w.rpc.Call(ctx, "getLatestBlockhash", params, &resp); err != nil {
		return solana.Hash{}, fmt.Errorf("getLatestBlockhash failed: %w", err)
	}

	if resp.Error != nil {
		return solana.Hash{}, fmt.Errorf("getLatestBlockhash error: %s", resp.Error.Message)
	}

	// Decode blockhash
	hash, err := solana.HashFromBase58(resp.Result.Value.Blockhash)
	if err != nil {
		return solana.Hash{}, fmt.Errorf("invalid blockhash format: %w", err)
	}

	return hash, nil
}

// SimulateTransaction simulates a transaction before sending
func (w *Wallet) SimulateTransaction(ctx context.Context, tx *solana.Transaction) (*SimulationResult, error) {
	// Serialize transaction
	txBytes, err := tx.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("failed to serialize transaction: %w", err)
	}

	encodedTx := base64.StdEncoding.EncodeToString(txBytes)

	var resp struct {
		Result struct {
			Value struct {
				Err      interface{} `json:"err"`
				Logs     []string    `json:"logs"`
				Accounts []struct {
					Executable bool   `json:"executable"`
					Owner      string `json:"owner"`
					Lamports   uint64 `json:"lamports"`
					Data       []any  `json:"data"`
				} `json:"accounts,omitempty"`
				UnitsConsumed uint64 `json:"unitsConsumed,omitempty"`
			} `json:"value"`
		} `json:"result"`
		Error *projectrpc.RPCError `json:"error"`
	}

	params := []any{
		encodedTx,
		map[string]any{
			"encoding":   "base64",
			"commitment": "processed",
		},
	}

	if err := w.rpc.Call(ctx, "simulateTransaction", params, &resp); err != nil {
		return nil, fmt.Errorf("simulateTransaction failed: %w", err)
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("simulateTransaction error: %s", resp.Error.Message)
	}

	result := &SimulationResult{
		Logs:          resp.Result.Value.Logs,
		UnitsConsumed: resp.Result.Value.UnitsConsumed,
	}

	// Check for simulation errors
	if resp.Result.Value.Err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("%v", resp.Result.Value.Err)
		return result, fmt.Errorf("simulation failed: %v", resp.Result.Value.Err)
	}

	result.Success = true
	return result, nil
}

// SimulationResult contains simulation output
type SimulationResult struct {
	Success       bool
	Error         string
	Logs          []string
	UnitsConsumed uint64
}

// ConfirmTransaction polls for transaction confirmation
func (w *Wallet) ConfirmTransaction(
	ctx context.Context,
	signature string,
	commitment string,
	timeout time.Duration,
) error {

	deadline := time.Now().Add(timeout)
	backoff := 500 * time.Millisecond
	maxBackoff := 4 * time.Second

	for time.Now().Before(deadline) {
		// Check signature status
		confirmed, err := w.checkSignatureStatus(ctx, signature, commitment)
		if err != nil {
			return fmt.Errorf("failed to check signature: %w", err)
		}

		if confirmed {
			return nil
		}

		// Exponential backoff
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		}
	}

	return fmt.Errorf("transaction confirmation timeout after %v", timeout)
}

// checkSignatureStatus checks if a signature is confirmed
func (w *Wallet) checkSignatureStatus(ctx context.Context, signature string, commitment string) (bool, error) {
	var resp struct {
		Result struct {
			Value []struct {
				Slot               uint64      `json:"slot"`
				Confirmations      *int        `json:"confirmations"`
				Err                interface{} `json:"err"`
				ConfirmationStatus string      `json:"confirmationStatus"`
			} `json:"value"`
		} `json:"result"`
		Error *projectrpc.RPCError `json:"error"`
	}

	params := []any{
		[]string{signature},
		map[string]any{"searchTransactionHistory": true},
	}

	if err := w.rpc.Call(ctx, "getSignatureStatuses", params, &resp); err != nil {
		return false, err
	}

	if resp.Error != nil {
		return false, fmt.Errorf("getSignatureStatuses error: %s", resp.Error.Message)
	}

	if len(resp.Result.Value) == 0 || resp.Result.Value[0].ConfirmationStatus == "" {
		return false, nil // Not yet processed
	}

	status := resp.Result.Value[0]

	// Check for transaction error
	if status.Err != nil {
		return false, fmt.Errorf("transaction failed: %v", status.Err)
	}

	// Check if commitment level is met
	switch commitment {
	case "processed":
		return status.ConfirmationStatus != "", nil
	case "confirmed":
		return status.ConfirmationStatus == "confirmed" || status.ConfirmationStatus == "finalized", nil
	case "finalized":
		return status.ConfirmationStatus == "finalized", nil
	default:
		return status.ConfirmationStatus != "", nil
	}
}

// BuildTransaction creates a new transaction with recent blockhash
func (w *Wallet) BuildTransaction(
	ctx context.Context,
	instructions []solana.Instruction,
) (*solana.Transaction, error) {

	// Get recent blockhash
	recentBlockhash, err := w.GetLatestBlockhash(ctx, "processed")
	if err != nil {
		return nil, fmt.Errorf("failed to get blockhash: %w", err)
	}

	// Create transaction
	tx, err := solana.NewTransaction(
		instructions,
		recentBlockhash,
		solana.TransactionPayer(w.pub),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	return tx, nil
}

// SignAndSend is a convenience method that builds, signs, and sends a transaction
func (w *Wallet) SignAndSend(
	ctx context.Context,
	instructions []solana.Instruction,
	opts *SendOptions,
) (string, error) {

	// Build transaction
	tx, err := w.BuildTransaction(ctx, instructions)
	if err != nil {
		return "", err
	}

	// Sign
	if err := w.SignTx(tx); err != nil {
		return "", err
	}

	// Send
	sig, err := w.SendTx(ctx, tx, opts)
	if err != nil {
		return "", err
	}

	return sig, nil
}
