package swapengine

import (
	"context"
	"fmt"
	"time"

	"github.com/aman-zulfiqar/solana-swap-indexer/internal/cache"
	"github.com/aman-zulfiqar/solana-swap-indexer/internal/models"
	"github.com/aman-zulfiqar/solana-swap-indexer/internal/orca"
	"github.com/aman-zulfiqar/solana-swap-indexer/internal/wallet"
	"github.com/gagliardetto/solana-go"
)

type TokenAccountResolver interface {
	Resolve(ctx context.Context, owner solana.PublicKey, mint solana.PublicKey) (*ResolvedTokenAccount, error)
}

type errTokenAccountResolver struct{}

func (errTokenAccountResolver) Resolve(ctx context.Context, owner solana.PublicKey, mint solana.PublicKey) (*ResolvedTokenAccount, error) {
	return nil, fmt.Errorf("token account resolution not implemented (need ATA + wSOL handling)")
}

type Executor struct {
	wallet       *wallet.Wallet
	orcaClient   *orca.Client
	poolRegistry *orca.PoolRegistry
	redis        *cache.RedisCache
	clickhouse   *cache.ClickHouseStore
	risk         *RiskManager

	tokenAccounts  TokenAccountResolver
	confirmTimeout time.Duration
}

func NewExecutor(
	w *wallet.Wallet,
	orcaClient *orca.Client,
	poolRegistry *orca.PoolRegistry,
	redis *cache.RedisCache,
	clickhouse *cache.ClickHouseStore,
	risk *RiskManager,
) *Executor {
	return &Executor{
		wallet:         w,
		orcaClient:     orcaClient,
		poolRegistry:   poolRegistry,
		redis:          redis,
		clickhouse:     clickhouse,
		risk:           risk,
		tokenAccounts:  errTokenAccountResolver{},
		confirmTimeout: 60 * time.Second,
	}
}

func (e *Executor) WithTokenAccountResolver(r TokenAccountResolver) *Executor {
	if r != nil {
		e.tokenAccounts = r
	}
	return e
}

func (e *Executor) GetQuote(ctx context.Context, params *SwapParams) (*QuoteResult, error) {
	if params == nil {
		return nil, fmt.Errorf("params is nil")
	}

	var pool *orca.LegacyPool
	var err error

	if params.PoolName != "" {
		pool, err = e.poolRegistry.FindPoolByName(params.PoolName)
	} else {
		pool, err = e.poolRegistry.FindPoolByMints(params.InputMint, params.OutputMint)
	}
	if err != nil {
		return nil, err
	}

	aToB, err := orca.DetermineSwapDirection(pool, params.InputMint)
	if err != nil {
		return nil, err
	}

	state, err := orca.RefreshPoolState(ctx, e.orcaClient, pool)
	if err != nil {
		return nil, err
	}

	reserveIn, reserveOut := state.GetReserves(aToB)

	amountOut, priceImpact, err := orca.CalculateLegacySwapOutput(
		params.AmountIn,
		reserveIn,
		reserveOut,
		pool.FeeNumerator,
		pool.FeeDenominator,
	)
	if err != nil {
		return nil, err
	}

	minOut := orca.ApplySlippage(amountOut, params.SlippageBps)
	params.MinAmountOut = minOut

	return &QuoteResult{
		PoolName:      pool.Name,
		AmountIn:      params.AmountIn,
		AmountOut:     amountOut,
		MinAmountOut:  minOut,
		PriceImpact:   priceImpact,
		FeeBps:        orca.CalculateFeeBps(pool.FeeNumerator, pool.FeeDenominator),
		ReserveIn:     reserveIn,
		ReserveOut:    reserveOut,
		ExecutionRate: float64(amountOut) / float64(params.AmountIn),
		QuotedAt:      time.Now(),
	}, nil
}

func (e *Executor) ExecuteSwap(ctx context.Context, params *SwapParams) (*SwapResult, error) {
	start := time.Now()

	quote, err := e.GetQuote(ctx, params)
	if err != nil {
		return &SwapResult{Success: false, Error: err.Error(), Quote: quote}, err
	}

	bal, err := e.wallet.GetBalanceSOL(ctx)
	if err != nil {
		return &SwapResult{Success: false, Error: err.Error(), Quote: quote}, err
	}

	riskCheck, err := e.risk.CheckSwap(ctx, params, quote, bal)
	if err != nil {
		return &SwapResult{Success: false, Error: err.Error(), Quote: quote}, err
	}
	if !riskCheck.Allowed {
		err := fmt.Errorf("risk check rejected: %s", riskCheck.Reason)
		return &SwapResult{Success: false, Error: err.Error(), Quote: quote}, err
	}

	// Pool lookup again (cheap) to build instruction
	var pool *orca.LegacyPool
	if params.PoolName != "" {
		pool, err = e.poolRegistry.FindPoolByName(params.PoolName)
	} else {
		pool, err = e.poolRegistry.FindPoolByMints(params.InputMint, params.OutputMint)
	}
	if err != nil {
		return &SwapResult{Success: false, Error: err.Error(), Quote: quote}, err
	}

	aToB, err := orca.DetermineSwapDirection(pool, params.InputMint)
	if err != nil {
		return &SwapResult{Success: false, Error: err.Error(), Quote: quote}, err
	}

	owner := e.wallet.PublicKey()

	if params.Intent == nil {
		return &SwapResult{Success: false, Error: "params.intent is nil", Quote: quote}, fmt.Errorf("params.intent is nil")
	}

	// Resolve token accounts (may add setup/cleanup instructions)
	inRes, err := e.tokenAccounts.Resolve(ctx, owner, params.InputMint)
	if err != nil {
		return &SwapResult{Success: false, Error: err.Error(), Quote: quote}, err
	}
	outRes, err := e.tokenAccounts.Resolve(ctx, owner, params.OutputMint)
	if err != nil {
		return &SwapResult{Success: false, Error: err.Error(), Quote: quote}, err
	}

	// Build pre/post instruction list
	var preIxs []solana.Instruction
	var postIxs []solana.Instruction
	preIxs = append(preIxs, inRes.PreIxs...)
	preIxs = append(preIxs, outRes.PreIxs...)

	// Wrap SOL input (TokenMints["SOL"] is the wSOL mint in this codebase)
	if params.InputMint.String() == TokenMints["SOL"] {
		preIxs = append(preIxs,
			NewSystemTransferIx(owner, inRes.Account, params.AmountIn),
			NewTokenSyncNativeIx(inRes.Account),
		)

		// If we created the wSOL ATA just for this swap, unwrap leftovers after swap
		// (safe cleanup for mock usage; avoids leaving rent + zero-balance accounts around)
		if inRes.Created && !params.OutputMint.Equals(params.InputMint) {
			postIxs = append(postIxs, NewTokenCloseAccountIx(inRes.Account, owner, owner))
		}
	}

	// If output is SOL (wSOL mint), optionally unwrap by closing the account ONLY if we created it in this tx.
	// This keeps behavior non-destructive for existing wallets.
	if params.OutputMint.String() == TokenMints["SOL"] && outRes.Created {
		postIxs = append(postIxs, NewTokenCloseAccountIx(outRes.Account, owner, owner))
	}

	ix, err := orca.BuildLegacySwapInstruction(
		pool,
		params.AmountIn,
		params.MinAmountOut,
		owner,
		inRes.Account,
		outRes.Account,
		aToB,
	)
	if err != nil {
		return &SwapResult{Success: false, Error: err.Error(), Quote: quote}, err
	}

	ixs := make([]solana.Instruction, 0, len(preIxs)+1+len(postIxs))
	ixs = append(ixs, preIxs...)
	ixs = append(ixs, ix)
	ixs = append(ixs, postIxs...)

	tx, err := e.wallet.BuildTransaction(ctx, ixs)
	if err != nil {
		return &SwapResult{Success: false, Error: err.Error(), Quote: quote}, err
	}

	if e.risk.config.RequireSimulation {
		if _, err := e.wallet.SimulateTransaction(ctx, tx); err != nil {
			return &SwapResult{Success: false, Error: err.Error(), Quote: quote}, err
		}
	}

	if err := e.wallet.SignTx(tx); err != nil {
		return &SwapResult{Success: false, Error: err.Error(), Quote: quote}, err
	}

	sig, err := e.wallet.SendTx(ctx, tx, nil)
	if err != nil {
		return &SwapResult{Success: false, Error: err.Error(), Quote: quote}, err
	}

	if err := e.wallet.ConfirmTransaction(ctx, sig, "confirmed", e.confirmTimeout); err != nil {
		return &SwapResult{Signature: sig, Success: false, Error: err.Error(), Quote: quote}, err
	}

	// publish to redis/clickhouse (best-effort)
	ev := &models.SwapEvent{
		Signature: sig,
		Timestamp: time.Now(),
		Pair:      fmt.Sprintf("%s-%s", params.Intent.InputToken, params.Intent.OutputToken),
		TokenIn:   params.Intent.InputToken,
		TokenOut:  params.Intent.OutputToken,
		AmountIn:  params.Intent.Amount,
		AmountOut: 0, // TODO: decode actual out from logs; MVP keeps 0
		Price:     0,
		Fee:       0,
		Pool:      quote.PoolName,
		Dex:       "Orca",
	}
	if e.redis != nil {
		_ = e.redis.AddRecentSwap(ctx, ev)
		_ = e.redis.PublishSwap(ctx, ev)
	}
	if e.clickhouse != nil {
		_ = e.clickhouse.InsertSwap(ctx, ev)
	}

	e.risk.RecordSwap(params, quote)

	return &SwapResult{
		ExecutionID: fmt.Sprintf("exec_%d", time.Now().UnixNano()),
		Signature:   sig,
		Success:     true,
		Duration:    time.Since(start),
		Quote:       quote,
	}, nil
}
