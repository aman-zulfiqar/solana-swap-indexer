# Setup Guide

Complete setup and usage guide for the Solana Swap Indexer.

---

## Installation

### 1. Clone Repository

```bash
git clone https://github.com/aman-zulfiqar/solana-swap-indexer.git
cd solana-swap-indexer
```

### 2. Install Go Dependencies

```bash
go mod download
```

This installs:
- `github.com/redis/go-redis/v9` - Redis client
- `github.com/ClickHouse/clickhouse-go/v2` - ClickHouse client
- `github.com/sirupsen/logrus` - Structured logging

### 3. Verify Build

```bash
go build ./...
```

---

## Infrastructure Setup

### Start Docker Services

```bash
# Start Redis + ClickHouse + Web UIs
docker-compose up -d

# Verify services are running
docker-compose ps
```

### Initialize ClickHouse Schema

The schema is auto-initialized via `init.sql`. To manually reinitialize:

```bash
# Restart ClickHouse (triggers init.sql)
docker-compose restart clickhouse

# Or execute manually
docker exec -it solana-clickhouse clickhouse-client \
  --queries-file /docker-entrypoint-initdb.d/init.sql
```

### Verify Infrastructure

```bash
# Test Redis
docker exec -it solana-redis redis-cli ping
# Expected: PONG

# Test ClickHouse
docker exec -it solana-clickhouse clickhouse-client -q "SELECT 1"
# Expected: 1
```

---

## Running the Indexer

### Basic Usage (Public RPC)

```bash
cd cmd/indexer
go run .
```

Expected output:
```
2024-12-24 15:30:00 level=info msg="connected to Redis" addr=localhost:6379
2024-12-24 15:30:00 level=info msg="connected to ClickHouse" addr=localhost:9000 database=solana
2024-12-24 15:30:00 level=info msg="starting Solana swap indexer" interval=30s provider=rpc rpc_url=https://api.mainnet-beta.solana.com
2024-12-24 15:30:00 level=info msg="starting RPC polling" interval=30s programs=[675kPX9MHTjS2zt1qfr1NYHuzeLXfQM9H24wFSUt1Mp8]
2024-12-24 15:30:00 level=info msg="indexer running, press Ctrl+C to stop"
```

### Build and Run Binary

```bash
# Build
go build -o indexer .

# Run
./indexer
```

---

---

## Viewing Data

### Option 1: Redis Commander (Web UI)

**URL**: http://localhost:8081

Features:
- Browse all Redis keys visually
- View `swaps:recent` list (last 100 swaps)
- View `price:*` keys (current token prices)
- No login required

Navigation:
1. Open http://localhost:8081
2. Click on `swaps:recent` in the left panel
3. View JSON swap data in the right panel

### Option 2: ClickHouse Tabix (Web UI)

**URL**: http://localhost:8080

Connection settings:
| Field | Value |
|-------|-------|
| Name | `Solana` (or any name) |
| URL | `http://localhost:8123` |
| Login | `default` |
| Password | *(leave empty)* |

Important: Do NOT use semicolons at the end of queries in Tabix.

Example queries (no semicolons):
```sql
SELECT * FROM solana.swaps ORDER BY timestamp DESC LIMIT 20

SELECT pair, count() as trades FROM solana.swaps GROUP BY pair ORDER BY trades DESC
```

### Option 3: ClickHouse HTTP API

Direct browser access:

**View latest swaps:**
```
http://localhost:8123/?query=SELECT * FROM solana.swaps ORDER BY timestamp DESC LIMIT 10&default_format=PrettyCompact
```

**Count total swaps:**
```
http://localhost:8123/?query=SELECT count() FROM solana.swaps&default_format=PrettyCompact
```

**JSON format:**
```
http://localhost:8123/?query=SELECT * FROM solana.swaps LIMIT 5&default_format=JSON
```

### Option 4: Command Line

**Redis CLI:**
```bash
docker exec -it solana-redis redis-cli

# View recent swaps
LRANGE swaps:recent 0 9

# Get token price
GET price:USDC

# Count cached swaps
LLEN swaps:recent

# List all keys
KEYS *
```

**ClickHouse CLI:**
```bash
docker exec -it solana-clickhouse clickhouse-client

# Query (semicolons OK in CLI)
SELECT * FROM solana.swaps ORDER BY timestamp DESC LIMIT 10;
```

---

## Query Examples

### ClickHouse Queries

**Recent swaps:**
```sql
SELECT 
    signature,
    timestamp,
    pair,
    amount_in,
    token_in,
    amount_out,
    token_out,
    price
FROM solana.swaps
ORDER BY timestamp DESC
LIMIT 20
```

**Volume by pair:**
```sql
SELECT 
    pair,
    count() as swap_count,
    sum(amount_out) as total_volume,
    avg(price) as avg_price,
    min(price) as min_price,
    max(price) as max_price
FROM solana.swaps
GROUP BY pair
ORDER BY total_volume DESC
LIMIT 10
```

**Hourly activity:**
```sql
SELECT 
    toStartOfHour(timestamp) as hour,
    count() as swaps,
    sum(amount_out) as volume
FROM solana.swaps
WHERE timestamp >= now() - INTERVAL 24 HOUR
GROUP BY hour
ORDER BY hour DESC
```

**Largest swaps:**
```sql
SELECT 
    signature,
    pair,
    amount_in,
    token_in,
    amount_out,
    token_out,
    price,
    timestamp
FROM solana.swaps
ORDER BY amount_out DESC
LIMIT 10
```

**Swaps by DEX:**
```sql
SELECT 
    dex,
    count() as swaps,
    sum(amount_out) as volume
FROM solana.swaps
GROUP BY dex
ORDER BY swaps DESC
```

**Price history for a pair:**
```sql
SELECT 
    toStartOfMinute(timestamp) as minute,
    avg(price) as price,
    count() as trades
FROM solana.swaps
WHERE pair = 'SOL/USDC'
  AND timestamp >= now() - INTERVAL 1 HOUR
GROUP BY minute
ORDER BY minute DESC
```

### Redis Commands

```bash
# Recent swaps (last 10)
LRANGE swaps:recent 0 9

# Recent swaps (last 50)
LRANGE swaps:recent 0 49

# Get all prices
KEYS price:*

# Get specific price
GET price:SOL
GET price:USDC

# Count cached swaps
LLEN swaps:recent

# Monitor live activity
MONITOR

# Clear all data (careful!)
FLUSHDB
```

---

## Troubleshooting

### Rate Limit Errors (429)

**Symptoms:**
```
level=error msg="poll error" error="max retries exceeded: rate limited (429)"
```

**Solutions:**
1. Increase poll interval:
   ```bash
   export POLL_INTERVAL=60s
   ```
2. Use Triton for higher limits:
   ```bash
   export STREAM_PROVIDER=triton
   export TRITON_API_KEY=your_key
   ```
3. Use a private RPC endpoint

### No Data in ClickHouse

**Check:**
1. Verify indexer is running and processing swaps
2. Check logs for insertion errors
3. Query directly:
   ```bash
   docker exec -it solana-clickhouse clickhouse-client -q "SELECT count() FROM solana.swaps"
   ```

### Redis Connection Failed

**Check:**
1. Verify Redis is running:
   ```bash
   docker exec -it solana-redis redis-cli ping
   ```
2. Check port 6379 is not in use:
   ```bash
   lsof -i :6379
   ```

### ClickHouse Connection Failed

**Check:**
1. Verify ClickHouse is running:
   ```bash
   docker exec -it solana-clickhouse clickhouse-client -q "SELECT 1"
   ```
2. Check port 9000 is not in use:
   ```bash
   lsof -i :9000
   ```

### Tabix UI Syntax Error

**Symptoms:**
```
Syntax error (Multi-statements are not allowed)
```

**Solution:**
Remove the semicolon from the end of your query in Tabix.

### Empty Swap Results

Some transactions are not swaps (liquidity adds, etc.). Check logs for:
```
level=debug msg="not a swap transaction (insufficient token balances)"
```

This is normal - the indexer filters non-swap transactions.

### Graceful Shutdown

Press `Ctrl+C` to stop the indexer gracefully. You should see:
```
level=info msg="shutting down gracefully"
level=info msg="closing connections"
```

---

## Stopping Services

### Stop Indexer

Press `Ctrl+C` in the terminal running the indexer.

### Stop Docker Services

```bash
# Stop containers (keep data)
docker-compose down

# Stop and remove all data
docker-compose down -v

# View logs before stopping
docker-compose logs -f
```

### Clean Restart

```bash
# Stop everything and remove data
docker-compose down -v

# Start fresh
docker-compose up -d

# Run indexer
cd cmd/indexer && go run .
```

---

# Solana Swap Indexer - Example Queries

## Basic Stats
- "Show the last 20 swaps with signature, pair, amount_in, amount_out, price."
- "Top 10 trading pairs by total amount_out in the last 24 hours."
- "Which tokens have the highest number of swaps in the last 7 days?"
- "Average price of SOL/USDC over the last 6 hours."
- "Give me min, max, and avg price for SOL/USDT today."

## Time-based Volume / Activity
- "Hourly swap count and average price for SOL/USDC in the last 24 hours."
- "Daily total USDC volume across all pairs for the last 7 days."
- "What was total SOL volume (amount_out in SOL) in the last 12 hours?"
- "Which hour today had the highest total swap volume?"

## DEX / Pool Comparisons
- "Compare total amount_out volume by dex in the last 24 hours."
- "For Raydium only, top 5 pairs by swap count in the last 3 days."
- "Which pool has the highest average price volatility in the last 48 hours?"

## Whale / Anomaly-ish Queries
- "Show the 20 largest swaps by amount_out in the last 24 hours."
- "Any swaps where price was more than 5x the average price for that pair in the last 7 days?"
- "Which pairs have extremely large amount_out values but very small price (possible rugs)?"
- "Find all swaps today where amount_out > 1,000,000 of any token."

## Pair-focused Questions
- "For pair SOL/USDC, show total volume, swap count, and average price in the last 24 hours."
- "For SOL/USDT, show a 1-hour time series of avg price and volume for the last 12 hours."
- "Which pairs had zero swaps in the last 24 hours but had swaps in the previous week?"

## Sanity / Debug Prompts
- "What’s the full schema of solana.swaps and a sample of 5 rows?"
- "List distinct pairs ordered by swap count descending."
- "Show 10 random swaps to inspect raw data: signature, pair, token_in, token_out, amount_in, amount_out, price."
