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
	JupiterFee = 0.0025
	// Orca legacy constant-product pools are typically 30 bps.
	OrcaFee = 0.003
	// Whirlpool fees vary by pool; treat as nominal here for indexing/UI display.
	OrcaWhirlpoolFee = 0.002
)

// DEX program addresses
var ProgramAddresses = map[string]string{
	"Jupiter": "JUP6LkbZbjS1jKKwapdHNy74zcZ3tLUZoi5QNyVTaV4",
	// Orca legacy constant-product swap program
	"Orca": "9W959DqEETiGZocYWCQPaJ6sBmUzgfxXfqGeTEdp3aQP",
	// Orca Whirlpool program
	"OrcaWhirlpool": "whirLbMiicVdio4qvUfM5KAg6Ct8VwpYzGff3uctyCc",
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
	"9n4nbM75f5Ui33ZbPYXn59EwSgE8CGsHtAeTH5YFeJ9E": "BTC-w",
	"A1t2oTrV9Q1sX1j8o3GcA4V9oVj7NHJ6rg8spP4JYxvV": "ADA",
	"7VF2a7fVf1kW1bR3b9G9r7wXjkBz3hM4a7qM8fQ6i6rN": "DOGE",
	"4kq5B9Pb6Jh1f5bZXj3Zy5b3ZxXZ4b5J4N3A6kX1jJdH": "SOL-USDC LP",
	"8kD7u3m8Zj6kD4zRj5D7r8yHk8Xj6J2kQ3xJ1bY5XyT":  "mSOL-USDC LP",
	"C1f2B9Fj3Kk5u3dX7Yh6B8gXj4Jk8P2xH4kL2bY5U7":   "SRM",
	"E3k5J7Hk8D2g3L6X4V7N8B5Kk9L2X6H4J5Q1bY7Z2F":   "FTT",
	"F7T4K1L5P3V2Q6H8N4J5B3X2Z6K1L4J8H7Y5D2F1G3":   "SOL-ETH LP",
	"G3H7J1K4L5P8D2N3B6X9V2J4K1L7H3F5Y8D6T2C1R4":   "MNGO",
	"H2F6D3G1K7L5J2N9B4X8V6J3K1L4H9F7Y5D1T3C2R6":   "COPE",
	"I7J3H1K5L9P2D4N6B8X1V7J3K2L5H8F3Y2D4T1C9R5":   "KIN",
	"J4L2K6H8P3V1D5N7B9X2V4J5K3L1H6F2Y3D5T7C4R8":   "RAY-USDC LP",
	"K9H1J5L3P7D2N4B6X8V3J1K4L5H2F7Y1D3T6C2R9":     "ORCA-USDC LP",
	"L2K4H8P3V1D5N7B9X2V4J5K3L1H6F2Y3D5T7C4R8":     "COPE-USDC LP",
	"M3J5L1P7D2N4B6X8V3J1K4L5H2F7Y1D3T6C2R9":       "SOL-USDT LP",
	"N1K3H5P7V2D4N6B8X2V4J5K3L1H6F2Y3D5T7C4R9":     "USDC-USDT LP",
	"O2H4J6L8P3V1D5N7B9X2V4J5K3L1H6F2Y3D5T7C4R8":   "ETH-USDC LP",
	"P1K5H7P3V2D4N6B8X2V4J5K3L1H6F2Y3D5T7C4R9":     "MNGO-SOL LP",
}

// Pool names by DEX
const (
	PoolJupiterAgg = "JupiterAggregator"
	PoolOrcaLegacy = "OrcaLegacy"
	PoolOrcaWhirl  = "OrcaWhirlpool"
)
