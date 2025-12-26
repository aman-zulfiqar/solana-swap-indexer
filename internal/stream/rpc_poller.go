package stream

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/aman-zulfiqar/solana-swap-indexer/internal/constants"
	"github.com/aman-zulfiqar/solana-swap-indexer/internal/models"
	"github.com/aman-zulfiqar/solana-swap-indexer/internal/rpc"
	"github.com/aman-zulfiqar/solana-swap-indexer/internal/storage"

	"github.com/sirupsen/logrus"
)

// RPCPoller implements StreamProvider for polling Solana RPC
type RPCPoller struct {
	client           *rpc.Client
	programAddresses []string
	pollInterval     time.Duration
	logger           *logrus.Logger

	mu            sync.RWMutex
	lastSignature string
	running       bool
}

// RPCPollerConfig holds configuration for the RPC poller
type RPCPollerConfig struct {
	RPCClient        *rpc.Client
	ProgramAddresses []string
	PollInterval     time.Duration
	Logger           *logrus.Logger
}

// NewRPCPoller creates a new RPC poller
func NewRPCPoller(cfg RPCPollerConfig) *RPCPoller {
	if cfg.Logger == nil {
		cfg.Logger = logrus.New()
	}

	if len(cfg.ProgramAddresses) == 0 {
		cfg.ProgramAddresses = []string{
			constants.ProgramAddresses["Raydium"],
		}
	}

	return &RPCPoller{
		client:           cfg.RPCClient,
		programAddresses: cfg.ProgramAddresses,
		pollInterval:     cfg.PollInterval,
		logger:           cfg.Logger,
	}
}

// Start begins polling for swap events
func (r *RPCPoller) Start(ctx context.Context, handler storage.SwapHandler) error {
	r.mu.Lock()
	if r.running {
		r.mu.Unlock()
		return fmt.Errorf("poller already running")
	}
	r.running = true
	r.mu.Unlock()

	ticker := time.NewTicker(r.pollInterval)
	defer ticker.Stop()

	r.logger.WithFields(logrus.Fields{
		"interval": r.pollInterval,
		"programs": r.programAddresses,
	}).Info("starting RPC polling")

	for {
		select {
		case <-ctx.Done():
			r.mu.Lock()
			r.running = false
			r.mu.Unlock()
			return ctx.Err()

		case <-ticker.C:
			if err := r.poll(ctx, handler); err != nil {
				r.logger.WithError(err).Error("poll error")
			}
		}
	}
}

// Stop stops the poller
func (r *RPCPoller) Stop() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.running = false
	return nil
}

// poll fetches and processes new transactions
func (r *RPCPoller) poll(ctx context.Context, handler storage.SwapHandler) error {
	opts := map[string]interface{}{
		"limit": constants.SignatureBatchSize,
	}

	r.mu.RLock()
	lastSig := r.lastSignature
	r.mu.RUnlock()

	if lastSig != "" {
		opts["until"] = lastSig
		r.logger.WithField("after", lastSig[:8]).Debug("fetching new signatures")
	}

	// Fetch signatures
	sigResp, err := r.client.GetSignaturesForAddress(ctx, r.programAddresses[0], opts)
	if err != nil {
		return fmt.Errorf("failed to get signatures: %w", err)
	}

	if len(sigResp.Result) == 0 {
		r.logger.Debug("no new transactions")
		return nil
	}

	r.logger.WithField("count", len(sigResp.Result)).Info("found new signatures")

	// Update last signature
	r.mu.Lock()
	r.lastSignature = sigResp.Result[0].Signature
	r.mu.Unlock()

	// Process each transaction with delay to avoid rate limits
	for i, sig := range sigResp.Result {
		if sig.Err != nil {
			r.logger.WithField("signature", sig.Signature[:8]).Debug("skipping failed transaction")
			continue
		}

		// Add delay between requests to avoid rate limiting
		if i > 0 {
			r.logger.WithField("delay", constants.DelayBetweenTxFetch).Debug("waiting before next request")
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(constants.DelayBetweenTxFetch):
			}
		}

		r.logger.WithFields(logrus.Fields{
			"index":     fmt.Sprintf("%d/%d", i+1, len(sigResp.Result)),
			"signature": sig.Signature[:8],
		}).Debug("processing transaction")

		swap, err := r.parseTransaction(ctx, sig.Signature, sig.BlockTime)
		if err != nil {
			r.logger.WithError(err).WithField("signature", sig.Signature[:8]).Warn("failed to parse transaction")
			continue
		}

		if swap != nil {
			handler(swap)
		}
	}

	return nil
}

// parseTransaction fetches and parses a transaction into a SwapEvent
func (r *RPCPoller) parseTransaction(ctx context.Context, signature string, blockTime int64) (*models.SwapEvent, error) {
	txResp, err := r.client.GetTransaction(ctx, signature)
	if err != nil {
		return nil, err
	}

	if txResp.Result == nil || txResp.Result.Meta == nil {
		return nil, fmt.Errorf("empty transaction result")
	}

	meta := txResp.Result.Meta

	if meta.Err != nil {
		return nil, fmt.Errorf("transaction failed")
	}

	// Need at least 2 token balance changes for a swap
	if len(meta.PreTokenBalances) < 2 || len(meta.PostTokenBalances) < 2 {
		r.logger.WithField("signature", signature[:8]).Debug("not a swap transaction (insufficient token balances)")
		return nil, nil
	}

	// Calculate balance changes
	balanceChanges := make(map[int]float64)
	for _, pre := range meta.PreTokenBalances {
		balanceChanges[pre.AccountIndex] = -pre.UITokenAmount.UIAmount
	}
	for _, post := range meta.PostTokenBalances {
		balanceChanges[post.AccountIndex] += post.UITokenAmount.UIAmount
	}

	// Collect non-zero changes
	var changes []rpc.BalanceChange
	for _, post := range meta.PostTokenBalances {
		change := balanceChanges[post.AccountIndex]
		if change != 0 {
			changes = append(changes, rpc.BalanceChange{
				Mint:   post.Mint,
				Amount: change,
			})
		}
	}

	if len(changes) < 2 {
		r.logger.WithField("signature", signature[:8]).Debug("not a swap transaction (no token changes)")
		return nil, nil
	}

	// Determine token in/out based on balance direction
	var tokenIn, tokenOut string
	var amountIn, amountOut float64

	for _, ch := range changes {
		if ch.Amount < 0 {
			amountIn = -ch.Amount
			tokenIn = r.getTokenSymbol(ch.Mint)
		} else if ch.Amount > 0 {
			amountOut = ch.Amount
			tokenOut = r.getTokenSymbol(ch.Mint)
		}
	}

	// Validate swap data
	if tokenIn == "" || tokenOut == "" || amountIn == 0 || amountOut == 0 {
		r.logger.WithField("signature", signature[:8]).Debug("could not parse swap details")
		return nil, nil
	}

	// Skip same-token conversions (e.g., wrapped SOL)
	if tokenIn == tokenOut {
		r.logger.WithField("signature", signature[:8]).Debug("skipping same-token conversion")
		return nil, nil
	}

	price := amountOut / amountIn
	pair := fmt.Sprintf("%s/%s", tokenIn, tokenOut)

	swap := &models.SwapEvent{
		Signature: signature,
		Timestamp: time.Unix(blockTime, 0),
		Pair:      pair,
		TokenIn:   tokenIn,
		TokenOut:  tokenOut,
		AmountIn:  amountIn,
		AmountOut: amountOut,
		Price:     price,
		Fee:       constants.RaydiumFee,
		Pool:      constants.PoolRaydiumAMM,
		Dex:       "Raydium",
	}

	r.logger.WithFields(logrus.Fields{
		"pair":       pair,
		"amount_in":  fmt.Sprintf("%.4f %s", amountIn, tokenIn),
		"amount_out": fmt.Sprintf("%.4f %s", amountOut, tokenOut),
		"price":      fmt.Sprintf("%.4f", price),
	}).Info("parsed swap")

	return swap, nil
}

// getTokenSymbol maps a token mint address to its symbol
func (r *RPCPoller) getTokenSymbol(mint string) string {
	if symbol, ok := constants.TokenSymbols[mint]; ok {
		return symbol
	}

	// Return shortened mint if unknown
	if len(mint) > 8 {
		return mint[:4] + "..." + mint[len(mint)-4:]
	}
	return mint
}
