package constants

import "time"

// Redis keys
const (
	RedisKeyRecentSwaps = "swaps:recent"
	RedisKeyPricePrefix = "price:"
)

// Redis Pub/Sub channels
const (
	PubSubChannelSwaps = "swaps:live"
)

// Limits
const (
	MaxRecentSwaps     = 100
	SignatureBatchSize = 3 // Reduced to avoid rate limits on public RPC
)

// Rate limiting
const (
	DelayBetweenTxFetch = 3 * time.Second // Delay between getTransaction calls
)

// DEX fees
const (
	RaydiumFee = 0.0025
	OrcaFee    = 0.003
	JupiterFee = 0.0025
)

// DEX program addresses
var ProgramAddresses = map[string]string{
	"Raydium": "675kPX9MHTjS2zt1qfr1NYHuzeLXfQM9H24wFSUt1Mp8",
	"Orca":    "whirLbMiicVdio4qvUfM5KAg6Ct8VwpYzGff3uctyCc",
	"Jupiter": "JUP6LkbZbjS1jKKwapdHNy74zcZ3tLUZoi5QNyVTaV4",
}

// Token mint addresses to symbols
var TokenSymbols = map[string]string{
	"So11111111111111111111111111111111111111112":  "SOL",
	"EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v": "USDC",
	"Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB": "USDT",
	"mSoLzYCxHdYgdzU16g5QSh3i5K3z3KZK7ytfqcJm7So":  "mSOL",
	"7vfCXTUXx5WJV5JADk17DUJ4ksgau7utNKj4b963voxs": "ETH",
	"3NZ9JMVBmGAqocybic2c7LQCJScmgsAZ6vQqTDzcqmJh": "BTC",
	"DezXAZ8z7PnrnRJjz3wXBoRgixCa6xjnB7YaB1pPB263": "BONK",
	"7GCihgDB8fe6KNjn2MYtkzZcRjQy3t9GHdC8uHYmW2hr": "POPCAT",
	"JUPyiwrYJFskUPiHa7hkeR8VUtAeFoSYbKedZNsDvCN":  "JUP",
	"4k3Dyjzvzp8eMZWUXbBCjEvwSkkk59S5iCNLY3QrkX6R": "RAY",
}

// Pool names by DEX
const (
	PoolRaydiumAMM = "RaydiumAMM"
	PoolOrcaWhirl  = "OrcaWhirlpool"
	PoolJupiterAgg = "JupiterAggregator"
)
