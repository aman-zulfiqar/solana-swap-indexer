# Solana Swap Indexer

Real-time indexer for tracking Solana DEX swaps with Redis caching, ClickHouse analytics, and Pub/Sub streaming.

## Features

- **Real-time swap tracking** from Raydium (extensible to Orca, Jupiter)
- **Redis Pub/Sub** for instant event broadcasting to multiple consumers
- **Multiple RPC providers**: Public RPC or Triton
- **Redis cache** for fast recent data access and token prices
- **ClickHouse** for long-term analytics and time-series queries
- **Structured logging** with logrus
- **Retry logic** with exponential backoff for rate limit handling
- **Graceful shutdown** with proper resource cleanup
- **Thread-safe** polling with mutex protection
- **Interface-based design** for testability and flexibility

## Architecture

```
Solana RPC
    │
    ▼
┌─────────────────┐
│   RPC Client    │  ← Retry + Timeout + Backoff
│  (rpc/client)   │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│   RPC Poller    │  ← Poll every 30s, parse token balances
│ (stream/poller) │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│    Indexer      │  ← Process swap events
│  (cmd/indexer)  │
└────────┬────────┘
         │
    ┌────┴────┬──────────────┐
    ▼         ▼              ▼
┌───────┐ ┌────────┐ ┌──────────────┐
│ Redis │ │ Click  │ │   PUBLISH    │
│ Cache │ │ House  │ │ "swaps:live" │
└───────┘ └────────┘ └──────┬───────┘
                            │
              ┌─────────────┼─────────────┐
              ▼             ▼             ▼
        ┌──────────┐ ┌──────────┐ ┌──────────┐
        │Subscriber│ │Alert Bot │ │WebSocket │
        │  (CLI)   │ │ (Future) │ │ (Future) │
        └──────────┘ └──────────┘ └──────────┘
```

## Project Structure

```
solana-swap-indexer/
├── cmd/
│   ├── indexer/main.go           # Application entry point
│   └── subscriber/main.go        # Real-time Pub/Sub viewer
├── internal/
│   ├── config/
│   │   └── config.go             # Environment configuration
│   ├── constants/
│   │   └── constants.go          # Named constants, token mappings
│   ├── rpc/
│   │   ├── client.go             # HTTP client with retry/timeout
│   │   └── types.go              # RPC request/response types
│   ├── storage/
│   │   └── interfaces.go         # SwapCache, SwapStore interfaces
│   ├── cache/
│   │   ├── redis.go              # Redis + Pub/Sub implementation
│   │   └── clickhouse.go         # ClickHouse implementation
│   ├── stream/
│   │   └── rpc_poller.go         # Transaction polling and parsing
│   └── models/
│       └── swap.go               # SwapEvent data model
├── docker-compose.yml            # Redis + ClickHouse infrastructure
├── init.sql                      # ClickHouse schema
└── go.mod
```

## Prerequisites

- Go 1.21+
- Docker & Docker Compose

## Quick Start

### 1. Install Dependencies

```bash
go mod download
```

### 2. Start Infrastructure

```bash
docker-compose up -d
docker-compose ps
```

Services:
| Service | Port | Description |
|---------|------|-------------|
| Redis | 6379 | Cache + Pub/Sub |
| ClickHouse | 9000 | Analytics DB (native) |
| ClickHouse | 8123 | Analytics DB (HTTP) |
| Redis Commander | 8081 | Redis Web UI |
| Tabix | 8080 | ClickHouse Web UI |

### 3. Run the Indexer

```bash
# Default: Public Solana RPC
go run cmd/indexer/main.go

# With Triton (higher rate limits)
STREAM_PROVIDER=triton TRITON_API_KEY=your_key go run cmd/indexer/main.go
```

### 4. Run the Live Viewer (Pub/Sub)

```bash
# In a separate terminal
go run cmd/subscriber/main.go
```

Output:
```
╔════════════════════════════════════════════════════════════════════════════════════════════════════════════╗
║                              Live Swap Viewer - Solana Swap Indexer (Pub/Sub)                              ║
╠════════════════════════════════════════════════════════════════════════════════════════════════════════════╣
║  Time     │ Pair                 │ Amount In              │ Amount Out             │ Price        │ Sig    ║
╚════════════════════════════════════════════════════════════════════════════════════════════════════════════╝
[20:11:47] MEW1...cPP5/SOL     │     100.4987 MEW1...cPP5 │       0.0007 SOL       │     0.000007 │ 4NPPUCL2
[20:11:29] AGzj...pump/SOL     │ 1148738.6167 AGzj...pump │       0.4074 SOL       │     0.000000 │ 6gY92yTm
```

## Configuration

All settings via environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `SOLANA_RPC_URL` | `https://api.mainnet-beta.solana.com` | Solana RPC endpoint |
| `STREAM_PROVIDER` | `rpc` | Provider: `rpc` or `triton` |
| `TRITON_API_KEY` | - | Triton API key |
| `POLL_INTERVAL` | `30s` | Polling interval |
| `REDIS_ADDR` | `localhost:6379` | Redis address |
| `CLICKHOUSE_ADDR` | `localhost:9000` | ClickHouse address |
| `CLICKHOUSE_DATABASE` | `solana` | ClickHouse database |
| `HTTP_TIMEOUT` | `30s` | HTTP request timeout |
| `MAX_RETRIES` | `3` | Max retry attempts |
| `RETRY_BACKOFF` | `1s` | Initial backoff duration |

## Data Storage

### Redis

Keys:
- `swaps:recent` - List of last 100 swaps (JSON)
- `price:{TOKEN}` - Current price (e.g., `price:SOL`)

Pub/Sub Channels:
- `swaps:live` - Real-time swap events in JSON format

```bash
# CLI access
docker exec -it solana-redis redis-cli
LRANGE swaps:recent 0 9
GET price:USDC

# Subscribe to live events
SUBSCRIBE swaps:live
```

### ClickHouse

Table: `solana.swaps`

```bash
# CLI access
docker exec -it solana-clickhouse clickhouse-client

# Query examples
SELECT * FROM swaps ORDER BY timestamp DESC LIMIT 10;
SELECT pair, count() as swaps FROM swaps GROUP BY pair ORDER BY swaps DESC;
```

HTTP access:
```
http://localhost:8123/?query=SELECT+*+FROM+swaps+LIMIT+10
```

## Redis Pub/Sub

The indexer broadcasts every swap event to a Redis Pub/Sub channel, enabling real-time consumers.

### Message Format

```json
{
  "signature": "5FW45cJU...",
  "timestamp": "2025-12-24T17:52:06Z",
  "pair": "SOL/USDC",
  "token_in": "SOL",
  "token_out": "USDC",
  "amount_in": 3.6154,
  "amount_out": 439.974,
  "price": 121.6948,
  "fee": 0.0025,
  "pool": "RaydiumAMM",
  "dex": "Raydium"
}
```

### Subscribe in Go Code

```go
package main

import (
    "context"
    "fmt"
    "github.com/aman-zulfiqar/solana-swap-indexer/internal/cache"
)

func main() {
    ctx := context.Background()
    
    redisCache, _ := cache.NewRedisCache(ctx, cache.RedisConfig{
        Addr: "localhost:6379",
    })
    defer redisCache.Close()

    // Subscribe to swaps
    swapChan, _ := redisCache.SubscribeSwaps(ctx)

    for swap := range swapChan {
        fmt.Printf("New swap: %s - %.4f %s -> %.4f %s\n",
            swap.Pair,
            swap.AmountIn, swap.TokenIn,
            swap.AmountOut, swap.TokenOut,
        )

        // React to specific conditions
        if swap.AmountOut > 10 && swap.TokenOut == "SOL" {
            fmt.Println("🚨 Whale alert!")
        }
    }
}
```

### Pub/Sub API

```go
// Publisher (Indexer Side)
func (r *RedisCache) PublishSwap(ctx context.Context, swap *models.SwapEvent) error

// Subscriber (Consumer Side)
func (r *RedisCache) SubscribeSwaps(ctx context.Context) (<-chan *models.SwapEvent, error)
```

### Pub/Sub Limitations

1. **No message persistence** - If no subscribers are listening, messages are lost
2. **No replay** - New subscribers don't receive historical messages
3. **No acknowledgment** - No guarantee of delivery

For persistent messaging with replay, consider using Redis Streams instead.

## Example Queries

### Recent Activity
```sql
SELECT 
    signature,
    pair,
    amount_in,
    token_in,
    amount_out,
    token_out,
    price
FROM swaps
ORDER BY timestamp DESC
LIMIT 20
```

### Volume by Pair
```sql
SELECT 
    pair,
    count() as swap_count,
    sum(amount_out) as total_volume,
    avg(price) as avg_price
FROM swaps
GROUP BY pair
ORDER BY total_volume DESC
LIMIT 10
```

### Hourly Stats
```sql
SELECT 
    toStartOfHour(timestamp) as hour,
    count() as swaps,
    avg(price) as avg_price
FROM swaps
WHERE timestamp >= now() - INTERVAL 24 HOUR
GROUP BY hour
ORDER BY hour DESC
```

## Web UIs

### Redis Commander
- URL: http://localhost:8081
- Navigate to `swaps:recent` to see cached swaps

### ClickHouse Tabix
- URL: http://localhost:8080
- Connection: `http://localhost:8123`
- Username: `default`
- Password: (empty)
- Note: Do not use semicolons at end of queries

## Logs

Sample output:
```
2024-12-24 15:30:00 level=info msg="connected to Redis" addr=localhost:6379
2024-12-24 15:30:00 level=info msg="connected to ClickHouse" addr=localhost:9000 database=solana
2024-12-24 15:30:00 level=info msg="starting Solana swap indexer" interval=30s provider=rpc
2024-12-24 15:30:30 level=info msg="found new signatures" count=5
2024-12-24 15:30:31 level=info msg="parsed swap" amount_in="1.5 SOL" amount_out="188.25 USDC" pair=SOL/USDC
2024-12-24 15:30:31 level=info msg="published swap to channel" signature=abcd1234 subscribers=2
2024-12-24 15:30:31 level=info msg="swap processed successfully" pair=SOL/USDC signature=abcd1234
```

## How It Works

1. **Polling**: Every 30s, fetch new transaction signatures from Raydium program
2. **Deduplication**: Track `lastSignature` to only fetch new transactions
3. **Transaction Fetch**: Call `getTransaction` for each signature
4. **Parsing**: Diff `preTokenBalances` vs `postTokenBalances` to calculate swap amounts
5. **Token Mapping**: Convert mint addresses to symbols (SOL, USDC, etc.)
6. **Storage**: Save to Redis (cache) and ClickHouse (analytics)
7. **Broadcast**: Publish to Redis Pub/Sub for real-time consumers

## Extending

### Add More DEXs

Edit `internal/constants/constants.go`:

```go
var ProgramAddresses = map[string]string{
    "Raydium": "675kPX9MHTjS2zt1qfr1NYHuzeLXfQM9H24wFSUt1Mp8",
    "Orca":    "whirLbMiicVdio4qvUfM5KAg6Ct8VwpYzGff3uctyCc",
    "Jupiter": "JUP6LkbZbjS1jKKwapdHNy74zcZ3tLUZoi5QNyVTaV4",
}
```

### Add More Tokens

Edit `internal/constants/constants.go`:

```go
var TokenSymbols = map[string]string{
    "So11111111111111111111111111111111111111112": "SOL",
    "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v": "USDC",
    // Add more...
}
```

### Add New Pub/Sub Consumers

Create a new subscriber service:

```go
// cmd/alerts/main.go
swapChan, _ := redisCache.SubscribeSwaps(ctx)

for swap := range swapChan {
    if swap.AmountOut > 100 && swap.TokenOut == "SOL" {
        sendTelegramAlert("🚨 Whale: " + swap.Signature)
    }
}
```

## Troubleshooting

**Rate limit errors (429)?**
- Increase `POLL_INTERVAL` to 60s or more
- Use Triton for higher limits
- Set custom `SOLANA_RPC_URL`

**No data in ClickHouse?**
- Check indexer logs for errors
- Verify ClickHouse: `docker exec -it solana-clickhouse clickhouse-client`
- Run `SELECT count() FROM swaps`

**Redis connection failed?**
- Verify: `docker exec -it solana-redis redis-cli ping`
- Check port 6379 availability

**Empty swap results?**
- Some transactions are not swaps (e.g., liquidity adds)
- Check logs for "not a swap transaction" messages

**Subscriber not receiving messages?**
- Ensure indexer is running (it publishes)
- Check Redis: `docker exec -it solana-redis redis-cli SUBSCRIBE swaps:live`

## Stopping

```bash
# Stop services
docker-compose down

# Stop and remove data
docker-compose down -v
```

## Future Enhancements

- [ ] WebSocket server for browser clients
- [ ] Telegram/Discord alert bot
- [ ] Filter subscriptions by pair or token
- [ ] Redis Streams for message persistence
- [ ] Whale detection alerts (> X SOL)
- [ ] LangChain agent for natural language queries

## License

MIT
