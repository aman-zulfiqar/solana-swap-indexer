package swapengine

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/aman-zulfiqar/solana-swap-indexer/internal/cache"
	"github.com/aman-zulfiqar/solana-swap-indexer/internal/orca"
	"github.com/aman-zulfiqar/solana-swap-indexer/internal/rpc"
	"github.com/aman-zulfiqar/solana-swap-indexer/internal/wallet"
)

// Engine is the main orchestrator for swap operations
type Engine struct {
	wallet         *wallet.Wallet
	orcaClient     *orca.Client
	poolRegistry   *orca.PoolRegistry
	redisCache     *cache.RedisCache
	clickhouse     *cache.ClickHouseStore
	decisionEngine *DecisionEngine
	executor       *Executor
	riskManager    *RiskManager
}

// EngineConfig holds configuration for the swap engine
type EngineConfig struct {
	// RPC settings
	RPCURL       string
	RPCTimeout   time.Duration
	MaxRetries   int
	RetryBackoff time.Duration

	// Wallet
	WalletPrivateKey string

	// Pool configuration
	PoolConfigPath string

	// Storage
	RedisAddr      string
	ClickHouseAddr string
	ClickHouseDB   string

	// Risk management
	RiskConfig RiskConfig
}

// DefaultEngineConfig returns sensible defaults
func DefaultEngineConfig() EngineConfig {
	return EngineConfig{
		RPCURL:         "https://api.mainnet-beta.solana.com",
		RPCTimeout:     30 * time.Second,
		MaxRetries:     3,
		RetryBackoff:   1 * time.Second,
		PoolConfigPath: "internal/config/pools.json",
		RedisAddr:      "",
		ClickHouseAddr: "",
		ClickHouseDB:   "",
		RiskConfig:     DefaultRiskConfig(),
	}
}

// NewEngine creates a new swap engine with all dependencies
func NewEngine(cfg EngineConfig) (*Engine, error) {
	// 1. Initialize wallet
	walletCfg := wallet.WalletConfig{
		RPCURL:              cfg.RPCURL,
		PrivateKey:          cfg.WalletPrivateKey,
		Timeout:             cfg.RPCTimeout,
		MaxRetries:          cfg.MaxRetries,
		RetryBackoff:        cfg.RetryBackoff,
		DefaultCommitment:   "confirmed",
		SkipPreflight:       false,
		PreflightCommitment: "processed",
	}

	w, err := wallet.NewWallet(walletCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create wallet: %w", err)
	}

	// 2. Initialize Orca client
	rpcCfg := rpc.ClientConfig{
		BaseURL:      cfg.RPCURL,
		Timeout:      cfg.RPCTimeout,
		MaxRetries:   cfg.MaxRetries,
		RetryBackoff: cfg.RetryBackoff,
	}

	orcaClient, err := orca.NewClient(rpcCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create Orca client: %w", err)
	}

	// 3. Load pool registry
	poolRegistry, err := orca.NewPoolRegistry(cfg.PoolConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load pool registry: %w", err)
	}

	// 4. Initialize Redis cache
	var redisCache *cache.RedisCache
	if cfg.RedisAddr != "" {
		rc, err := cache.NewRedisCache(context.Background(), cache.RedisConfig{Addr: cfg.RedisAddr})
		if err != nil {
			return nil, fmt.Errorf("failed to connect to Redis: %w", err)
		}
		redisCache = rc
	}

	// 5. Initialize ClickHouse
	var clickhouseStore *cache.ClickHouseStore
	if cfg.ClickHouseAddr != "" && cfg.ClickHouseDB != "" {
		ch, err := cache.NewClickHouseStore(context.Background(), cache.ClickHouseConfig{
			Addr:     cfg.ClickHouseAddr,
			Database: cfg.ClickHouseDB,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to connect to ClickHouse: %w", err)
		}
		clickhouseStore = ch
	}

	// 6. Create decision engine
	decisionEngine := NewDecisionEngine(cfg.RiskConfig)

	// 7. Create risk manager
	riskManager := NewRiskManager(cfg.RiskConfig)

	// 8. Create executor
	executor := NewExecutor(
		w,
		orcaClient,
		poolRegistry,
		redisCache,
		clickhouseStore,
		riskManager,
	).WithTokenAccountResolver(NewDefaultTokenAccountResolver(w))

	return &Engine{
		wallet:         w,
		orcaClient:     orcaClient,
		poolRegistry:   poolRegistry,
		redisCache:     redisCache,
		clickhouse:     clickhouseStore,
		decisionEngine: decisionEngine,
		executor:       executor,
		riskManager:    riskManager,
	}, nil
}

// NewEngineFromEnv creates an engine using environment variables
func NewEngineFromEnv() (*Engine, error) {
	cfg := DefaultEngineConfig()

	if v := os.Getenv("SOLANA_RPC_URL"); v != "" {
		cfg.RPCURL = v
	}
	cfg.WalletPrivateKey = os.Getenv("WALLET_PRIVATE_KEY")

	if v := os.Getenv("SWAPENGINE_POOL_CONFIG_PATH"); v != "" {
		cfg.PoolConfigPath = v
	}
	if v := os.Getenv("REDIS_ADDR"); v != "" {
		cfg.RedisAddr = v
	}
	if v := os.Getenv("CLICKHOUSE_ADDR"); v != "" {
		cfg.ClickHouseAddr = v
	}
	if v := os.Getenv("CLICKHOUSE_DATABASE"); v != "" {
		cfg.ClickHouseDB = v
	}

	if v := os.Getenv("SWAPENGINE_REQUIRE_SIMULATION"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			cfg.RiskConfig.RequireSimulation = b
		}
	}

	return NewEngine(cfg)
}

// ExecuteAISwap processes an AI-generated swap intent end-to-end
func (e *Engine) ExecuteAISwap(ctx context.Context, intent *SwapIntent) (*SwapResult, error) {
	// 1. Validate intent
	if err := e.decisionEngine.ValidateIntent(intent); err != nil {
		return nil, fmt.Errorf("invalid intent: %w", err)
	}

	// 2. Enrich with defaults
	e.decisionEngine.EnrichIntent(intent)

	// 3. Parse into executable parameters
	params, err := e.decisionEngine.ParseIntent(intent)
	if err != nil {
		return nil, fmt.Errorf("failed to parse intent: %w", err)
	}

	// 4. Execute the swap
	result, err := e.executor.ExecuteSwap(ctx, params)
	if err != nil {
		return result, fmt.Errorf("execution failed: %w", err)
	}

	return result, nil
}

// GetQuote returns a quote for a swap intent without executing
func (e *Engine) GetQuote(ctx context.Context, intent *SwapIntent) (*QuoteResult, error) {
	// Validate and parse
	if err := e.decisionEngine.ValidateIntent(intent); err != nil {
		return nil, fmt.Errorf("invalid intent: %w", err)
	}

	e.decisionEngine.EnrichIntent(intent)

	params, err := e.decisionEngine.ParseIntent(intent)
	if err != nil {
		return nil, fmt.Errorf("failed to parse intent: %w", err)
	}

	// Get quote
	return e.executor.GetQuote(ctx, params)
}

// CheckRisk validates a swap intent against risk rules without executing
func (e *Engine) CheckRisk(ctx context.Context, intent *SwapIntent) (*RiskCheckResult, error) {
	// Parse intent
	params, err := e.decisionEngine.ParseIntent(intent)
	if err != nil {
		return nil, fmt.Errorf("failed to parse intent: %w", err)
	}

	// Get quote
	quote, err := e.executor.GetQuote(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get quote: %w", err)
	}

	// Get wallet balance
	balance, err := e.wallet.GetBalanceSOL(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get balance: %w", err)
	}

	// Check risk
	return e.riskManager.CheckSwap(ctx, params, quote, balance)
}

// GetWalletInfo returns wallet status
func (e *Engine) GetWalletInfo(ctx context.Context) (*WalletInfo, error) {
	balance, err := e.wallet.GetBalanceSOL(ctx)
	if err != nil {
		return nil, err
	}

	return &WalletInfo{
		Address:    e.wallet.Address(),
		BalanceSOL: balance,
	}, nil
}

// GetPoolInfo returns information about available pools
func (e *Engine) GetPoolInfo() *PoolInfo {
	pools := e.poolRegistry.GetAllPools()
	poolNames := make([]string, len(pools))
	for i, pool := range pools {
		poolNames[i] = pool.Name
	}

	return &PoolInfo{
		TotalPools: len(pools),
		PoolNames:  poolNames,
	}
}

// GetRiskStatus returns current risk limits and usage
func (e *Engine) GetRiskStatus() *RiskStatus {
	dailyUsage := e.riskManager.dailyTracker.GetDailyUsage()

	return &RiskStatus{
		MaxSwapAmountSOL:  e.riskManager.config.MaxSwapAmountSOL,
		DailyLimitSOL:     e.riskManager.config.DailyLimitSOL,
		DailyUsedSOL:      dailyUsage,
		DailyRemainingSOL: e.riskManager.config.DailyLimitSOL - dailyUsage,
		AllowedTokens:     e.riskManager.config.AllowedTokens,
	}
}

// Close cleans up all resources
func (e *Engine) Close() error {
	var errs []error

	if err := e.wallet.Close(); err != nil {
		errs = append(errs, fmt.Errorf("wallet close: %w", err))
	}

	if err := e.orcaClient.Close(); err != nil {
		errs = append(errs, fmt.Errorf("orca client close: %w", err))
	}

	if e.redisCache != nil {
		if err := e.redisCache.Close(); err != nil {
			errs = append(errs, fmt.Errorf("redis close: %w", err))
		}
	}

	if e.clickhouse != nil {
		if err := e.clickhouse.Close(); err != nil {
			errs = append(errs, fmt.Errorf("clickhouse close: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("close errors: %v", errs)
	}

	return nil
}

// Info types
type WalletInfo struct {
	Address    string
	BalanceSOL float64
}

type PoolInfo struct {
	TotalPools int
	PoolNames  []string
}

type RiskStatus struct {
	MaxSwapAmountSOL  float64
	DailyLimitSOL     float64
	DailyUsedSOL      float64
	DailyRemainingSOL float64
	AllowedTokens     []string
}
