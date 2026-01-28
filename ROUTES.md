# Postman Test Guide — Solana Swap Indexer API (Echo)

Base URL:
- `http://localhost:8090`

All endpoints are versioned under:
- `http://localhost:8090/v1`

---

## 0) Start the API locally

### A) Run the API
From the repo root:

```bash
go run cmd/api/main.go
```

Env vars you may want:

- `API_ADDR=:8090`
- `API_KEY=...` (optional; if set you must send `X-API-Key`)
- `DEV=true` (enables `details` field in JSON errors)

Note on defaults:
- `API_ADDR` defaults to `:8090`
- `API_KEY` now has a default value (so auth is optional unless you override it)
- `DEV` defaults to `true` (so you will see `details` in errors by default)

### B) Redis/ClickHouse note
Most endpoints need **Redis**; `/v1/ai/ask` needs **ClickHouse** + `OPENROUTER_API_KEY`.

You ran `docker compose up` and got:

- `Cannot connect to the Docker daemon ... Is the docker daemon running?`

So Redis/ClickHouse may NOT be running yet.

If Docker is not available right now, you can still test:
- `GET /v1/health`
- `POST /v1/echo`

But swaps/prices/flags require Redis.

---

## 1) Common Postman setup

Create a Postman environment:
- `baseUrl` = `http://localhost:8090`
- `apiKey` = `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`

Required headers (use `{{apiKey}}` in Postman):
- `X-API-Key: {{apiKey}}`

All requests below assume:
- `Content-Type: application/json`

---

## Quick curl commands (for direct testing)

If you prefer curl over Postman, here are the exact commands that work:

```bash
#start with
lsof -ti:8090 | xargs kill -9 2>/dev/null || true
API_KEY=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 go run main.go
# Health
curl -s -X GET "http://localhost:8090/v1/health" -H "X-API-Key: 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" | jq .

# Echo
curl -s -X POST "http://localhost:8090/v1/echo" -H "Content-Type: application/json" -H "X-API-Key: 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" -d '{"hello": "world"}' | jq .

# Flags CRUD
curl -s -X POST "http://localhost:8090/v1/flags" -H "Content-Type: application/json" -H "X-API-Key: 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" -d '{"key": "test.flag", "value": true}' | jq .
curl -s -X GET "http://localhost:8090/v1/flags/test.flag" -H "X-API-Key: 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" | jq .
curl -s -X GET "http://localhost:8090/v1/flags" -H "X-API-Key: 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" | jq .
curl -s -X PUT "http://localhost:8090/v1/flags/test.flag" -H "Content-Type: application/json" -H "X-API-Key: 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" -d '{"value": false}' | jq .
curl -s -X DELETE "http://localhost:8090/v1/flags/test.flag" -H "X-API-Key: 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" -w "\n%{http_code}\n"

# Swaps
curl -s -X GET "http://localhost:8090/v1/swaps/recent?limit=5" -H "X-API-Key: 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" | jq .

# Prices
curl -s -X GET "http://localhost:8090/v1/prices/SOL" -H "X-API-Key: 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" | jq .

# AI Ask
curl -s -X POST "http://localhost:8090/v1/ai/ask" -H "Content-Type: application/json" -H "X-API-Key: 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" -d '{"question": "What were the top 5 pairs by total amount_out in the last 24 hours?"}' | jq .
curl -s -X POST "http://localhost:8090/v1/ai/ask" -H "Content-Type: application/json" -H "X-API-Key: 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" -d '{"question": "What is the average price for SOL/USDC"}' | jq .
```

---

## 2) Health

### Request
- Method: `GET`
- URL: `{{baseUrl}}/v1/health`
- Headers:
  - `X-API-Key: {{apiKey}}`

### Expected response
```json
{ "ok": true }
```

---

## 3) Echo (connectivity test)

### Request
- Method: `POST`
- URL: `{{baseUrl}}/v1/echo`
- Headers:
  - `X-API-Key: {{apiKey}}`
- Body:
```json
{ "hello": "world", "timestamp": 1234567890 }
```

### Expected response
Should return the same JSON body.

---

## 4) Flags (Redis required)

Redis keys used:
- `flags:index`
- `flags:{key}`

### 4.1 Upsert flag

- Method: `POST`
- URL: `{{baseUrl}}/v1/flags`
- Headers:
  - `X-API-Key: {{apiKey}}`
- Body:
```json
{ "key": "agent.repl", "value": true }
```

Expected response (shape):
```json
{ "key": "agent.repl", "value": true, "updated_at": "2026-01-05T...Z" }
```

### 4.2 Get flag

- Method: `GET`
- URL: `{{baseUrl}}/v1/flags/agent.repl`
- Headers:
  - `X-API-Key: {{apiKey}}`

### 4.3 Update flag

- Method: `PUT`
- URL: `{{baseUrl}}/v1/flags/agent.repl`
- Headers:
  - `X-API-Key: {{apiKey}}`
- Body:
```json
{ "value": false }
```

### 4.4 List flags

- Method: `GET`
- URL: `{{baseUrl}}/v1/flags`
- Headers:
  - `X-API-Key: {{apiKey}}`

Expected response:
```json
{ "items": [ { "key": "agent.repl", "value": false, "updated_at": "..." } ] }
```

### 4.5 Delete flag

- Method: `DELETE`
- URL: `{{baseUrl}}/v1/flags/agent.repl`
- Headers:
  - `X-API-Key: {{apiKey}}`

Expected response:
- Status: `204 No Content`

---

## 5) Swaps (Redis required)

### 5.1 Recent swaps

- Method: `GET`
- URL: `{{baseUrl}}/v1/swaps/recent?limit=20`
- Headers:
  - `X-API-Key: {{apiKey}}`

Validation rules:
- `limit` must be an integer
- `1 <= limit <= 200`

Expected response:
```json
{ "items": [ { "signature": "...", "pair": "SOL/USDC", "amount_in": 1.23, "amount_out": 456.7, "token_in": "SOL", "token_out": "USDC" } ] }
```

---

## 6) Prices (Redis required)

### 6.1 Get token price

- Method: `GET`
- URL: `{{baseUrl}}/v1/prices/SOL`
- Headers:
  - `X-API-Key: {{apiKey}}`

Expected response:
```json
{ "token": "SOL", "price": 123.45 }
```

Notes:
- Token is normalized to uppercase.
- If no price is set yet, you may see `price: 0`.

---

## 7) AI Ask (ClickHouse + OpenRouter required)

Requirements:
- ClickHouse must be reachable (`CLICKHOUSE_ADDR`, `CLICKHOUSE_DATABASE`, etc.)
- `OPENROUTER_API_KEY` must be set

Rate limiting:
- This endpoint is throttled (basic limiter). If you spam requests you may get `429`.

### 7.1 Ask (default model)

- Method: `POST`
- URL: `{{baseUrl}}/v1/ai/ask`
- Headers:
  - `X-API-Key: {{apiKey}}`
- Body:
```json
{ "question": "What were the top 5 pairs by total amount_out in the last 24 hours?" }
```

### Expected response
```json
{
  "sql": "SELECT\n    pair,\n    SUM(amount_out) AS total_amount_out\nFROM solana.swaps\nWHERE timestamp >= now() - INTERVAL 24 HOUR\nGROUP BY pair\nORDER BY total_amount_out DESC\nLIMIT 5",
  "answer": "The top 5 pairs by total amount_out in the last 24 hours are:\n\n- SOL/14DQ...35m7: approximately 2.21 million\n- USD1...EmuB/14DQ...35m7: approximately 558 thousand\n- SOL/MEW1...cPP5: approximately 401 thousand\n- SOL/2umQ...moon: approximately 102 thousand\n- SOL/BP8R...BAPo: approximately 79 thousand",
  "took_ms": 4045
}
```

### 7.2 Ask (override model)

- Method: `POST`
- URL: `{{baseUrl}}/v1/ai/ask`
- Headers:
  - `X-API-Key: {{apiKey}}`
- Body:
```json
{ "question": "Show the average price for SOL/USDC over the last 6 hours.", "model": "openai/gpt-4.1-mini" }
```

### Expected response (may fail with NaN if no data in time range)
```json
{
  "error": "ai ask failed",
  "code": 500,
  "details": {
    "err": "failed to marshal rows to JSON: json: unsupported value: NaN"
  }
}
```

### 7.3 Alternative working query (if above fails)

- Method: `POST`
- URL: `{{baseUrl}}/v1/ai/ask`
- Headers:
  - `X-API-Key: {{apiKey}}`
- Body:
```json
{ "question": "What is the average price for SOL/USDC" }
```

### Expected response
```json
{
  "sql": "SELECT avg(price) AS average_price\nFROM solana.swaps\nWHERE pair = 'SOL/USDC'",
  "answer": "- The average price for the SOL/USDC pair is approximately $188.02.",
  "took_ms": 2057
}
```

---

## 8) Error responses (what to expect)

All errors are JSON:

```json
{ "error": "message", "code": 400 }
```

If `DEV=true`, you may also see:

```json
{ "error": "message", "code": 400, "details": { "field": "..." } }
```

Common examples:
- Invalid limit:
```json
{ "error": "invalid limit", "code": 400, "details": { "limit": "min 1 max 200" } }
```
- Missing flag:
```json
{ "error": "flag not found", "code": 404 }
```

---

## 9) Quick sanity run order

1. `GET {{baseUrl}}/v1/health`
2. `POST {{baseUrl}}/v1/echo`
3. Start Redis (Docker Desktop / local redis)
4. `POST {{baseUrl}}/v1/flags` (create)
5. `GET {{baseUrl}}/v1/flags` (list)
6. `GET {{baseUrl}}/v1/swaps/recent?limit=5`
7. Start ClickHouse + set `OPENROUTER_API_KEY`
8. `POST {{baseUrl}}/v1/ai/ask`

---

## 10) Quote (Jupiter proxy)

### Request
- Method: `GET`
- URL: `{{baseUrl}}/v1/quote`
- Headers:
  - `X-API-Key: {{apiKey}}`

Required query params:
- `inputMint` (string)
- `outputMint` (string)
- `amount` (uint64 raw, before decimals)

Optional:
- `slippageBps` (uint16)
- `swapMode` (ExactIn|ExactOut)
- `dexes` (comma-separated) e.g. `dexes=Raydium,Orca%2BV2`
- `excludeDexes` (comma-separated)
- `restrictIntermediateTokens` (bool)
- `onlyDirectRoutes` (bool)
- `asLegacyTransaction` (bool)
- `platformFeeBps` (uint16)
- `maxAccounts` (uint64)
- `instructionVersion` (V1|V2)
- `dynamicSlippage` (bool)

### Quick curl

```bash
curl -s "http://localhost:8090/v1/quote?inputMint=So11111111111111111111111111111111111111112&outputMint=EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v&amount=100000000&slippageBps=50&swapMode=ExactIn" \
  -H "X-API-Key: YOUR_API_KEY" | jq .
```

Notes:
- This endpoint proxies Jupiter `GET /swap/v1/quote`.
- If you want Jupiter API key auth, set `JUPITER_API_KEY` in your env.
- To hit preprod, set `JUPITER_BASE_URL=https://preprod-quote-api.jup.ag`.
