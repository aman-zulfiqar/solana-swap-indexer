# 🚀 Solana Swap Indexer

Real-time indexer for tracking Solana DEX swaps with Redis caching, ClickHouse analytics, and Pub/Sub distribution.

## ⚡ Features

- **Real-time swap tracking** from Raydium, Orca, Jupiter, and other Solana DEXs
- **Multiple data sources**: Helius WebSocket, RPC polling, or Triton
- **Redis cache** for fast recent data access
- **ClickHouse** for long-term analytics and time-series queries
- **Pub/Sub** for live event distribution to subscribers
- **Rate limit handling** with automatic backoff
- **Docker-based** infrastructure (Redis + ClickHouse)

## 📁 Project Structure

```
solana-swap-indexer/
├── cmd/
│   ├── indexer/main.go      # Main indexer service
│   └── subscriber/main.go   # Example Pub/Sub subscriber
├── internal/
│   ├── stream/
│   │   ├── helius.go        # Helius WebSocket client
│   │   └── rpc_poller.go    # Free RPC polling
│   ├── cache/
│   │   ├── redis.go         # Redis cache layer
│   │   ├── pubsub.go        # Redis Pub/Sub wrapper
│   │   └── clickhouse.go    # ClickHouse storage
│   └── models/
│       └── swap.go          # Swap data model
├── docker-compose.yml       # Infrastructure (Redis + ClickHouse)
├── init.sql                 # ClickHouse schema
└── go.mod                   # Go dependencies
```

## 🛠️ Setup

### 1. Prerequisites

- Go 1.21+
- Docker & Docker Compose

### 2. Install Dependencies

```bash
go mod download
```

### 3. Start Infrastructure

```bash
# Start Redis + ClickHouse
docker-compose up -d

# Verify services
docker-compose ps
```

Services running:
- **Redis**: `localhost:6379`
- **ClickHouse**: `localhost:9000` (native), `localhost:8123` (HTTP)
- **Redis Commander UI**: http://localhost:8081
- **Tabix ClickHouse UI**: http://localhost:8080

### 4. Run the Indexer

**Option A: Free Public RPC (default)**
```bash
go run cmd/indexer/main.go
```

**Option B: Helius (recommended - free 100k credits/month)**
```bash
# Sign up at https://helius.dev
STREAM_PROVIDER=helius HELIUS_API_KEY=your_key_here go run cmd/indexer/main.go
```

**Option C: Triton**
```bash
# Sign up at https://triton.one
STREAM_PROVIDER=triton TRITON_API_KEY=your_key_here go run cmd/indexer/main.go
```

### 5. Run a Subscriber (optional)

```bash
# In a separate terminal
go run cmd/subscriber/main.go
```

## 📊 Viewing Data

### ClickHouse (SQL queries)

```bash
# Connect via CLI
docker exec -it solana-clickhouse clickhouse-client

# Query recent swaps
SELECT * FROM solana.swaps ORDER BY timestamp DESC LIMIT 10;

# Count by DEX
SELECT dex, count() as total FROM solana.swaps GROUP BY dex;

# View hourly aggregations
SELECT * FROM solana.swaps_hourly ORDER BY hour DESC LIMIT 10;
```

**Or via HTTP:**
```
http://localhost:8123/?query=SELECT * FROM solana.swaps LIMIT 10
```

### Redis (cache data)

```bash
# Connect via CLI
docker exec -it solana-redis redis-cli

# View recent swaps
LRANGE swaps:recent 0 9

# Get price
GET price:USDC

# Monitor live activity
MONITOR
```

**Or via Web UI:**
http://localhost:8081 (Redis Commander)

## 🔍 Example Queries

### ClickHouse Analytics

```sql
-- Top trading pairs by volume
SELECT 
    pair,
    count() as swap_count,
    sum(amount_out) as total_volume
FROM solana.swaps
GROUP BY pair
ORDER BY total_volume DESC
LIMIT 10;

-- Price statistics per DEX
SELECT 
    dex,
    pair,
    avg(price) as avg_price,
    min(price) as min_price,
    max(price) as max_price
FROM solana.swaps
WHERE timestamp >= now() - INTERVAL 1 HOUR
GROUP BY dex, pair;

-- Hourly swap activity
SELECT 
    hour,
    sum(swap_count) as total_swaps,
    avg(avg_price) as price
FROM solana.swaps_hourly
WHERE hour >= now() - INTERVAL 24 HOUR
GROUP BY hour
ORDER BY hour DESC;
```

### Redis Commands

```bash
# Get last 50 swaps
LRANGE swaps:recent 0 49

# Get current price
GET price:USDC
GET price:SOL

# Subscribe to live swaps
SUBSCRIBE swaps:all

# Subscribe to specific pair
SUBSCRIBE swaps:pair:SOL/USDC

# Pattern subscribe
PSUBSCRIBE swaps:pair:*
```

## 📡 Pub/Sub Channels

The indexer publishes to multiple Redis channels:

- `swaps:all` - All swap events
- `swaps:pair:{PAIR}` - Pair-specific (e.g., `swaps:pair:SOL/USDC`)
- `swaps:dex:{DEX}` - DEX-specific (e.g., `swaps:dex:Raydium`)
- `price:updates` - Price feed updates

## 🎯 Configuration

Environment variables:

```bash
# Stream provider: helius, rpc, or triton
STREAM_PROVIDER=rpc

# Helius API key (if using helius)
HELIUS_API_KEY=your_key

# Triton API key (if using triton)
TRITON_API_KEY=your_key

# Custom RPC URL (if using rpc provider)
SOLANA_RPC_URL=https://api.mainnet-beta.solana.com
```

## 🚦 Rate Limits

Public Solana RPC is rate-limited. The indexer:
- Polls every **10 seconds** by default
- Tracks `lastSignature` to avoid duplicates
- Handles 429 errors gracefully with backoff
- Logs skipped failed transactions

For production, use Helius or Triton for higher limits.

## 🛑 Stopping

```bash
# Stop services
docker-compose down

# Stop and remove data
docker-compose down -v
```

## 🧪 Testing Pub/Sub

```bash
# Terminal 1: Subscribe
docker exec -it solana-redis redis-cli
SUBSCRIBE swaps:all

# Terminal 2: Run indexer
go run cmd/indexer/main.go

# You'll see live swaps in Terminal 1!
```

## 📈 Monitoring

Check indexer logs for:
- `🔄 Starting RPC polling` - Poller started
- `📥 Found N new signatures` - New transactions discovered
- `⏭️ Skipping failed tx` - Failed transactions filtered
- `🔍 Processing tx` - Transaction being processed
- `✅ Swap processed successfully` - Saved to Redis + ClickHouse
- `❌ RPC error: code 429` - Rate limit hit

## 🔧 Troubleshooting

**No data in ClickHouse?**
- Check indexer logs for errors
- Verify ClickHouse is running: `docker-compose ps`
- Test connection: `docker exec -it solana-clickhouse clickhouse-client`

**Rate limit errors?**
- Slow down polling (increase `pollInterval` in `rpc_poller.go`)
- Switch to Helius/Triton for higher limits
- Use custom RPC endpoint

**Redis connection failed?**
- Verify Redis is running: `docker exec -it solana-redis redis-cli ping`
- Check port 6379 is not in use

## 📝 Next Steps

1. **Parse real swap data** - Currently using mock amounts; decode Raydium instructions
2. **Add more DEXs** - Orca, Jupiter, Meteora, etc.
3. **WebSocket API** - Real-time feed for frontends
4. **Alerts** - Price alerts, volume spikes
5. **Analytics dashboard** - Grafana + ClickHouse

## 📄 License

MIT

## 🤝 Contributing

PRs welcome! Focus areas:
- Instruction parsing for different DEX programs
- More data sources (gRPC, Yellowstone)
- Performance optimizations
- Additional analytics queries

---

Built with ❤️ for the Solana ecosystem

