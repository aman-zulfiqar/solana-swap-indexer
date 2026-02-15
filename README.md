# Solana Swap Indexer & Execution Engine

A comprehensive platform for **tracking**, **analyzing**, and **automating** Solana DEX swaps. It combines real-time indexing with an AI-driven execution engine and a modern data dashboard.

## Features

### 🔍 Indexing & Analytics
- **Real-time Tracking**: Monitors swaps on Raydium, Orca, and Jupiter.
- **High-Performance Storage**: Uses Redis for hot caching and ClickHouse for historical analytics.
- **Pub/Sub Streaming**: Broadcasts live swap events via Redis Channels.
- **Data Dashboard**: Next.js-based UI for exploring swap data and controlling the system.

### 🤖 AI & Execution (SwapEngine)
- **AI-Driven Intents**: Parses natural language (e.g., "Buy 5 SOL worth of USDC") into executable swaps.
- **Risk Management**: Enforces daily limits, per-transaction limits, and token whitelists.
- **Automated Execution**: Handles quoting, slippage protection, transaction simulation, and confirmation.
- **Safe**: Built-in validation layers before any transaction is signed.

## Architecture

```
                                  ┌───────────────┐
                                  │   User / UI   │
                                  └───────┬───────┘
                                          │
                                          ▼
┌─────────────────┐       ┌─────────────────────────────────┐
│  Solana RPC     │       │           API Server            │
│ (Public/Triton) │◄──────┤       (cmd/api/main.go)         │
└───────┬─────────┘       └───────┬───────────────┬─────────┘
        │                         │               │
        ▼                         ▼               ▼
┌─────────────────┐       ┌──────────────┐ ┌──────────────┐
│   RPC Poller    │       │   AI Agent   │ │  SwapEngine  │
│ (stream/poller) │       │ (LangChain)  │ │ (Automated)  │
└───────┬─────────┘       └───────┬──────┘ └──────┬───────┘
        │                         │               │
        ▼                         ▼               ▼
┌─────────────────┐       ┌─────────────────────────────────┐
│     Indexer     │       │      Redis (Cache & Pub/Sub)    │
│  (cmd/indexer)  │──────►│    ClickHouse (Analytics DB)    │
└─────────────────┘       └─────────────────────────────────┘
```

## Project Structure

```
solana-swap-indexer/
├── cmd/
│   ├── indexer/          # Main data collector & processor
│   ├── swapengine/       # AI-driven execution engine
│   ├── ai-agent/         # LLM query interface
│   ├── api/              # REST API server
│   └── subscriber/       # CLI Pub/Sub listener
├── internal/
│   ├── swapengine/       # Core execution logic (Risk, Decision, Executor)
│   ├── orca/             # Orca DEX integration
│   ├── jupiter/          # Jupiter aggregator integration
│   ├── wallet/           # Key management & signing
│   ├── cache/            # Redis & ClickHouse adapters
│   └── models/           # Data structs
├── data-explorer-dashboard/ # Next.js Frontend
├── docker-compose.yml    # Infrastructure (Redis, ClickHouse)
└── init.sql              # Database Schema
```

## Prerequisites

- **Go** 1.21+
- **Docker** & Docker Compose
- **Node.js** 18+ (for Dashboard)
- **Solana Wallet** (for SwapEngine execution)

## Quick Start

### 1. Environment Setup

Create a `.env` file in the root directory. You **must** set these variables:

```bash
# Infrastructure
REDIS_ADDR=localhost:6379
CLICKHOUSE_ADDR=localhost:9000
CLICKHOUSE_DATABASE=solana
CLICKHOUSE_USERNAME=default
CLICKHOUSE_PASSWORD=your_password_here

# Solana
SOLANA_RPC_URL=https://api.mainnet-beta.solana.com
POLL_INTERVAL=30s
# Set 'triton' if using Triton RPC, else 'rpc'
STREAM_PROVIDER=rpc
TRITON_API_KEY=

# Execution (Required for SwapEngine)
WALLET_PRIVATE_KEY=your_base58_private_key_here

# API & Security
API_ADDR=:8080
API_KEY=secret-api-key
DEV=true

# AI / LLM
OPENROUTER_API_KEY=sk-or-your-key

# Resilience
HTTP_TIMEOUT=30s
MAX_RETRIES=3
RETRY_BACKOFF=1s
```

### 2. Start Infrastructure

Start Redis, ClickHouse, and management UIs:

```bash
docker-compose up -d
```

### 3. Run Services

You can run each service in a separate terminal:

**A. Indexer** (Collects Data)
```bash
go run cmd/indexer/main.go
```

**B. API Server** (Backend for UI)
```bash
go run cmd/api/main.go
```

**C. Swap Engine** (Optional - if executing trades)
```bash
go run cmd/swapengine/main.go
```

### 4. Start Dashboard

```bash
cd data-explorer-dashboard
pnpm install  # or npm install
pnpm dev      # or npm run dev
```

Visit the dashboard at `http://localhost:3000`.

## Configuration Reference

| Category        | Variable             | Description |
|-----------------|----------------------|-------------|
| **Solana**      | `SOLANA_RPC_URL`     | Mainnet/Testnet RPC Endpoint |
|                 | `POLL_INTERVAL`      | Frequency of indexer polling (e.g. `30s`) |
| **Storage**     | `REDIS_ADDR`         | Redis connection string |
|                 | `CLICKHOUSE_ADDR`    | ClickHouse native port (`9000`) |
| **SwapEngine**  | `WALLET_PRIVATE_KEY` | Private key for signing transactions |
| **AI**          | `OPENROUTER_API_KEY` | API Key for LLM reasoning |
| **API**         | `API_ADDR`           | Port for the Go API server |
|                 | `API_KEY`            | Simple auth key for API requests |

## Component Details

### Indexer
The backbone of the system. It polls the Solana blockchain for transactions involving known DEX program IDs (Raydium, Orca, etc.), parses the token balance changes to determine swap amounts, and stores the normalized data.

### Swap Engine
An automated trading system documented fully in [SWAPENGINE.md](SWAPENGINE.md).
- **Decision Engine**: Validates intents.
- **Risk Manager**: Checks balance, allowances, and slippage.
- **Executor**: Interacts with the chain.

### Data Dashboard
A web interface to:
- View live swap feeds.
- Query historical data using AI ("Show me top volume tokens").
- Manage feature flags.

## Troubleshooting

- **Panic: missing required environment variable**: Ensure your `.env` file contains ALL variables listed in the "Quick Start" section. The system enforces strict config validation.
- **Connection Refused**: Check if Docker containers are running (`docker-compose ps`).
- **Rate Limits**: If using public RPC, increase `POLL_INTERVAL` to `60s` or use a paid provider like Triton or Helius.

## License

MIT
