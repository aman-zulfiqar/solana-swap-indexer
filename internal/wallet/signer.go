package wallet

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	projectrpc "github.com/aman-zulfiqar/solana-swap-indexer/internal/rpc"
	"github.com/gagliardetto/solana-go"
	"github.com/mr-tron/base58"
)

type WalletConfig struct {
	RPCURL       string
	Timeout      time.Duration
	MaxRetries   int
	RetryBackoff time.Duration

	PrivateKey string // base58-encoded 64-byte key OR solana-keygen JSON array

	DefaultCommitment   string // e.g. "confirmed"
	SkipPreflight       bool
	PreflightCommitment string // e.g. "processed"
}

type Wallet struct {
	cfg  WalletConfig
	rpc  *projectrpc.Client
	priv solana.PrivateKey
	pub  solana.PublicKey
}

func NewWallet(cfg WalletConfig) (*Wallet, error) {
	if cfg.RPCURL == "" {
		return nil, fmt.Errorf("wallet: RPCURL is required")
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}
	if cfg.MaxRetries == 0 {
		cfg.MaxRetries = 3
	}
	if cfg.RetryBackoff == 0 {
		cfg.RetryBackoff = 1 * time.Second
	}
	if cfg.DefaultCommitment == "" {
		cfg.DefaultCommitment = "confirmed"
	}
	if cfg.PreflightCommitment == "" {
		cfg.PreflightCommitment = "processed"
	}
	if strings.TrimSpace(cfg.PrivateKey) == "" {
		return nil, fmt.Errorf("wallet: PrivateKey is required")
	}

	priv, err := parsePrivateKey(cfg.PrivateKey)
	if err != nil {
		return nil, err
	}

	rpcClient := projectrpc.NewClient(projectrpc.ClientConfig{
		BaseURL:      cfg.RPCURL,
		Timeout:      cfg.Timeout,
		MaxRetries:   cfg.MaxRetries,
		RetryBackoff: cfg.RetryBackoff,
	})

	pub := priv.PublicKey()

	return &Wallet{
		cfg:  cfg,
		rpc:  rpcClient,
		priv: priv,
		pub:  pub,
	}, nil
}

func NewWalletFromEnv() (*Wallet, error) {
	cfg := WalletConfig{
		RPCURL:            os.Getenv("SOLANA_RPC_URL"),
		PrivateKey:        os.Getenv("WALLET_PRIVATE_KEY"),
		DefaultCommitment: os.Getenv("WALLET_COMMITMENT"),
	}
	return NewWallet(cfg)
}

func (w *Wallet) Address() string             { return w.pub.String() }
func (w *Wallet) PublicKey() solana.PublicKey { return w.pub }
func (w *Wallet) Close() error                { return nil }

func (w *Wallet) GetBalanceSOL(ctx context.Context) (float64, error) {
	var resp struct {
		Result struct {
			Value uint64 `json:"value"` // lamports
		} `json:"result"`
		Error *projectrpc.RPCError `json:"error"`
	}

	params := []any{
		w.pub.String(),
		map[string]any{"commitment": w.cfg.DefaultCommitment},
	}

	if err := w.rpc.Call(ctx, "getBalance", params, &resp); err != nil {
		return 0, fmt.Errorf("getBalance RPC failed: %w", err)
	}
	if resp.Error != nil {
		return 0, fmt.Errorf("getBalance error: %s", resp.Error.Message)
	}

	return float64(resp.Result.Value) / 1e9, nil
}

// AccountExists checks if an account exists on-chain (getAccountInfo != nil).
func (w *Wallet) AccountExists(ctx context.Context, pubkey solana.PublicKey) (bool, error) {
	var resp struct {
		Result struct {
			Value any `json:"value"`
		} `json:"result"`
		Error *projectrpc.RPCError `json:"error"`
	}

	params := []any{
		pubkey.String(),
		map[string]any{
			"encoding":   "base64",
			"commitment": w.cfg.DefaultCommitment,
		},
	}

	if err := w.rpc.Call(ctx, "getAccountInfo", params, &resp); err != nil {
		return false, fmt.Errorf("getAccountInfo RPC failed: %w", err)
	}
	if resp.Error != nil {
		return false, fmt.Errorf("getAccountInfo error: %s", resp.Error.Message)
	}
	return resp.Result.Value != nil, nil
}

func parsePrivateKey(s string) (solana.PrivateKey, error) {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "[") {
		var ints []int
		if err := json.Unmarshal([]byte(s), &ints); err != nil {
			return nil, fmt.Errorf("wallet: invalid JSON private key: %w", err)
		}
		b := make([]byte, len(ints))
		for i, v := range ints {
			if v < 0 || v > 255 {
				return nil, fmt.Errorf("wallet: invalid byte at %d: %d", i, v)
			}
			b[i] = byte(v)
		}
		if len(b) != ed25519.PrivateKeySize {
			return nil, fmt.Errorf("wallet: expected %d bytes, got %d", ed25519.PrivateKeySize, len(b))
		}
		return solana.PrivateKey(ed25519.PrivateKey(b)), nil
	}

	raw, err := base58.Decode(s)
	if err != nil {
		return nil, fmt.Errorf("wallet: invalid base58 private key: %w", err)
	}
	if len(raw) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("wallet: expected %d bytes, got %d", ed25519.PrivateKeySize, len(raw))
	}
	return solana.PrivateKey(ed25519.PrivateKey(raw)), nil
}
