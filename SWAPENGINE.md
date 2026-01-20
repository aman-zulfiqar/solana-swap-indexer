# SwapEngine - AI-Driven Swap Execution

Automated swap execution engine with risk management, quote calculation, and transaction confirmation.

## Features

- ✅ **AI Intent Parsing** - Convert natural language intents to executable swaps
- ✅ **Risk Management** - Per-transaction and daily limits, token whitelisting
- ✅ **Quote Calculation** - Accurate constant-product AMM quotes with slippage
- ✅ **Transaction Simulation** - Test before sending
- ✅ **Automatic Confirmation** - Poll with exponential backoff
- ✅ **Redis/ClickHouse Integration** - Automatic execution logging
- ✅ **Comprehensive Error Handling** - Detailed failure reporting

## Architecture

```
AI Agent
    ↓
SwapIntent
    ↓
DecisionEngine (parse + validate)
    ↓
SwapParams
    ↓
RiskManager (check limits)
    ↓
Executor (quote → simulate → sign → send → confirm)
    ↓
SwapResult + Publish to Redis/ClickHouse
```

## Quick Start

### 1. Initialize Engine

```go
engine, err := swapengine.NewEngineFromEnv()
if err != nil {
    log.Fatal(err)
}
defer engine.Close()
```

### 2. Create Swap Intent

```go
intent := &swapengine.SwapIntent{
    InputToken:  "SOL",
    OutputToken: "USDC",
    Amount:      0.5, // 0.5 SOL
    Reason:      "Market conditions favorable",
    Confidence:  0.85,
}
```

### 3. Execute Swap

```go
ctx := context.Background()
result, err := engine.ExecuteAISwap(ctx, intent)

if result.Success {
    fmt.Printf("Swap successful! Signature: %s\n", result.Signature)
} else {
    fmt.Printf("Swap failed: %s\n", result.Error)
}
```

## Environment Variables

```bash
# Required
SOLANA_RPC_URL=https://api.testnet.solana.com
WALLET_PRIVATE_KEY=your_base58_key

# Optional (with defaults)
WALLET_COMMITMENT=confirmed
REDIS_ADDR=localhost:6379
CLICKHOUSE_ADDR=localhost:9000
CLICKHOUSE_DATABASE=solana
```

## Configuration

### Risk Settings

```go
cfg := swapengine.DefaultEngineConfig()

// Customize risk limits
cfg.RiskConfig.MaxSwapAmountSOL = 1.0    // Max 1 SOL per swap
cfg.RiskConfig.DailyLimitSOL = 10.0      // Max 10 SOL per day
cfg.RiskConfig.MaxPriceImpactBps = 500   // Max 5% price impact
cfg.RiskConfig.DefaultSlippageBps = 100  // 1% default slippage
cfg.RiskConfig.AllowedTokens = []string{"SOL", "USDC"}

engine, err := swapengine.NewEngine(cfg)
```

## Core Components

### 1. DecisionEngine (`decision.go`)

Converts AI intents to executable parameters:

```go
de := swapengine.NewDecisionEngine(riskConfig)

// Parse intent
params, err := de.ParseIntent(intent)

// Validate
err = de.ValidateIntent(intent)

// Enrich with defaults
de.EnrichIntent(intent)
```

### 2. RiskManager (`risk.go`)

Enforces safety limits:

```go
rm := swapengine.NewRiskManager(riskConfig)

// Check swap against all rules
riskCheck, err := rm.CheckSwap(ctx, params, quote, walletBalance)

if !riskCheck.Allowed {
    log.Printf("Rejected: %s", riskCheck.Reason)
}

// Record successful swap
rm.RecordSwap(params, quote)
```

**Risk Rules**:
- Per-transaction amount limits
- Rolling 24-hour daily limits
- Token whitelist enforcement
- Price impact thresholds
- Minimum balance requirements
- Slippage validation

### 3. Executor (`executor.go`)

Handles transaction execution:

```go
executor := swapengine.NewExecutor(
    wallet,
    orcaClient,
    poolRegistry,
    redisCache,
    clickhouse,
    riskManager,
)

// Execute swap
result, err := executor.ExecuteSwap(ctx, params)

// Get quote only
quote, err := executor.GetQuote(ctx, params)
```

**Execution Flow**:
1. Find pool by token pair
2. Refresh pool reserves
3. Calculate quote
4. Apply slippage
5. Check risk rules
6. Verify token accounts exist
7. Build swap instruction
8. Build transaction
9. Simulate
10. Sign
11. Send
12. Confirm (with polling)
13. Publish to Redis/ClickHouse
14. Update risk tracker

### 4. Engine (`engine.go`)

Main orchestrator:

```go
engine := swapengine.NewEngine(cfg)

// Execute AI swap
result, err := engine.ExecuteAISwap(ctx, intent)

// Get quote
quote, err := engine.GetQuote(ctx, intent)

// Check risk
riskCheck, err := engine.CheckRisk(ctx, intent)

// Get status
walletInfo, _ := engine.GetWalletInfo(ctx)
poolInfo := engine.GetPoolInfo()
riskStatus := engine.GetRiskStatus()
```

## Data Types

### SwapIntent (AI Input)

```go
type SwapIntent struct {
    InputToken        string   // "SOL", "USDC", etc.
    OutputToken       string
    Amount            float64  // Human-readable (e.g., 1.5 SOL)
    SlippageBps       *uint16  // Optional: 100 = 1%
    MaxPriceImpactBps *uint16  // Optional: 300 = 3%
    Reason            string   // AI reasoning
    Confidence        float64  // 0-1
    RequestedAt       time.Time
}
```

### SwapParams (Parsed)

```go
type SwapParams struct {
    InputMint         solana.PublicKey
    OutputMint        solana.PublicKey
    AmountIn          uint64  // Raw amount with decimals
    MinAmountOut      uint64  // With slippage
    PoolName          string
    SlippageBps       uint16
    MaxPriceImpactBps uint16
    Intent            *SwapIntent
    ParsedAt          time.Time
    ValidUntil        time.Time
}
```

### SwapResult (Output)

```go
type SwapResult struct {
    ExecutionID    string
    Signature      string
    Success        bool
    Error          string
    ExpectedOut    uint64
    ActualOut      *uint64
    Duration       time.Duration
    SimulationMS   int64
    ConfirmationMS int64
    Quote          *QuoteResult
    Execution      *SwapExecution
}
```

## Error Handling

All functions return detailed errors:

```go
result, err := engine.ExecuteAISwap(ctx, intent)
if err != nil {
    switch {
    case strings.Contains(err.Error(), "pool not found"):
        // No pool for this pair
    case strings.Contains(err.Error(), "risk check rejected"):
        // Failed risk validation
    case strings.Contains(err.Error(), "simulation failed"):
        // Transaction would fail
    case strings.Contains(err.Error(), "confirmation timeout"):
        // Sent but not confirmed
    default:
        // Other error
    }
}

// Result also contains detailed execution info
if !result.Success {
    fmt.Printf("Error: %s\n", result.Error)
    fmt.Printf("Logs:\n%v\n", result.Execution.Logs)
}
```

## Testing

```bash
# Unit tests
go test ./internal/swapengine -v

# Integration tests (requires testnet setup)
go test ./internal/swapengine -tags=integration -v

# Run specific test
go test ./internal/swapengine -run TestRiskManager -v
```

## Examples

See `examples_test.go` for comprehensive usage examples:

1. Simple swap execution
2. Quote-only request
3. Risk validation
4. Custom parameters
5. Status monitoring
6. Complete flow with error handling

## Security Considerations

### Never in Production Without:

1. **Rate Limiting** - Limit API calls and swaps per minute
2. **Real Oracle Integration** - Fetch current prices for risk calculations
3. **Multi-sig Wallet** - Don't use single-key wallets for large amounts
4. **Monitoring & Alerts** - Track failed swaps, high slippage, etc.
5. **Audit Logs** - Permanent record of all swap decisions
6. **Testnet Testing** - Thoroughly test on testnet first
7. **Mainnet Dry Run** - Simulate-only mode before live trading

### Current Limitations (MVP):

- Single wallet (no multi-sig)
- In-memory daily limits (resets on restart)
- Simplified price impact calculation
- No price oracle integration
- No MEV protection
- No transaction retry logic for failed confirms

## Troubleshooting

### Swap Fails with "pool not found"

```bash
# Check your pools.json config
cat configs/pools.json

# Ensure mints match
grep -A5 "SOL-USDC" configs/pools.json
```

### "Risk check rejected"

```go
// Check current limits
riskStatus := engine.GetRiskStatus()
fmt.Printf("Daily used: %.4f / %.4f SOL\n", 
    riskStatus.DailyUsedSOL, riskStatus.DailyLimitSOL)
```

### Simulation fails

```go
// Check logs in result
if !result.Success && result.Execution.SimulationOK {
    for _, log := range result.Execution.Logs {
        fmt.Println(log)
    }
}
```

### Confirmation timeout

Increase timeout or check RPC health:

```go
// In wallet.go, adjust confirmation timeout
err = wallet.ConfirmTransaction(ctx, sig, "confirmed", 90*time.Second)
```

## Future Enhancements

- [ ] Jupiter aggregator integration
- [ ] Multi-hop routing
- [ ] MEV protection (Jito integration)
- [ ] Advanced order types (limit, stop-loss)
- [ ] Portfolio rebalancing
- [ ] Gas optimization strategies
- [ ] Real-time price oracle
- [ ] Persistent daily limit tracking (Redis)
- [ ] Webhook notifications
- [ ] Grafana metrics dashboard

## Contributing

When adding features:

1. Update risk rules in `risk.go`
2. Add validation in `decision.go`
3. Extend execution logic in `executor.go`
4. Update examples and tests
5. Document in this README

## License

MIT