package orca

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/gagliardetto/solana-go"
)

// LegacyPoolConfig represents a pool entry in the JSON config
type LegacyPoolConfig struct {
	Name           string `json:"name"`
	ProgramID      string `json:"program_id"`
	SwapAccount    string `json:"swap_account"`
	Authority      string `json:"authority"`
	TokenMintA     string `json:"token_mint_a"`
	TokenMintB     string `json:"token_mint_b"`
	VaultA         string `json:"vault_a"`
	VaultB         string `json:"vault_b"`
	PoolMint       string `json:"pool_mint"`
	FeeAccount     string `json:"fee_account"`
	HostFeeAccount string `json:"host_fee_account,omitempty"`
	FeeNumerator   uint64 `json:"fee_numerator"`
	FeeDenominator uint64 `json:"fee_denominator"`
}

// LegacyPool represents a parsed, ready-to-use pool configuration
type LegacyPool struct {
	Name           string
	ProgramID      solana.PublicKey
	SwapAccount    solana.PublicKey
	Authority      solana.PublicKey
	TokenMintA     solana.PublicKey
	TokenMintB     solana.PublicKey
	VaultA         solana.PublicKey
	VaultB         solana.PublicKey
	PoolMint       solana.PublicKey
	FeeAccount     solana.PublicKey
	HostFeeAccount *solana.PublicKey
	FeeNumerator   uint64
	FeeDenominator uint64
}

// PoolRegistry holds all configured pools
type PoolRegistry struct {
	pools []LegacyPool
}

// NewPoolRegistry loads pools from a JSON file
func NewPoolRegistry(configPath string) (*PoolRegistry, error) {
	pools, err := LoadLegacyPoolsFromJSON(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load pools: %w", err)
	}

	return &PoolRegistry{
		pools: pools,
	}, nil
}

// LoadLegacyPoolsFromJSON reads and parses pool configurations
func LoadLegacyPoolsFromJSON(path string) ([]LegacyPool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var configs []LegacyPoolConfig
	if err := json.Unmarshal(data, &configs); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	pools := make([]LegacyPool, 0, len(configs))
	for i, cfg := range configs {
		pool, err := parsePoolConfig(cfg)
		if err != nil {
			return nil, fmt.Errorf("pool %d (%s): %w", i, cfg.Name, err)
		}
		pools = append(pools, pool)
	}

	return pools, nil
}

// parsePoolConfig converts a config struct to a LegacyPool with validation
func parsePoolConfig(cfg LegacyPoolConfig) (LegacyPool, error) {
	if cfg.FeeDenominator == 0 {
		return LegacyPool{}, fmt.Errorf("fee_denominator must be > 0")
	}

	pool := LegacyPool{
		Name:           cfg.Name,
		ProgramID:      solana.MustPublicKeyFromBase58(cfg.ProgramID),
		SwapAccount:    solana.MustPublicKeyFromBase58(cfg.SwapAccount),
		Authority:      solana.MustPublicKeyFromBase58(cfg.Authority),
		TokenMintA:     solana.MustPublicKeyFromBase58(cfg.TokenMintA),
		TokenMintB:     solana.MustPublicKeyFromBase58(cfg.TokenMintB),
		VaultA:         solana.MustPublicKeyFromBase58(cfg.VaultA),
		VaultB:         solana.MustPublicKeyFromBase58(cfg.VaultB),
		PoolMint:       solana.MustPublicKeyFromBase58(cfg.PoolMint),
		FeeAccount:     solana.MustPublicKeyFromBase58(cfg.FeeAccount),
		FeeNumerator:   cfg.FeeNumerator,
		FeeDenominator: cfg.FeeDenominator,
	}

	// Parse optional host fee account
	if cfg.HostFeeAccount != "" {
		hostFee := solana.MustPublicKeyFromBase58(cfg.HostFeeAccount)
		pool.HostFeeAccount = &hostFee
	}

	return pool, nil
}

// FindPoolByMints searches for a pool matching the given token pair
func (r *PoolRegistry) FindPoolByMints(
	mintA, mintB solana.PublicKey,
) (*LegacyPool, error) {

	for i := range r.pools {
		pool := &r.pools[i]

		// Check both directions: A->B and B->A
		if (pool.TokenMintA.Equals(mintA) && pool.TokenMintB.Equals(mintB)) ||
			(pool.TokenMintA.Equals(mintB) && pool.TokenMintB.Equals(mintA)) {
			return pool, nil
		}
	}

	return nil, fmt.Errorf("no pool found for mints %s / %s", mintA, mintB)
}

// FindPoolByName searches for a pool by its name
func (r *PoolRegistry) FindPoolByName(name string) (*LegacyPool, error) {
	for i := range r.pools {
		if r.pools[i].Name == name {
			return &r.pools[i], nil
		}
	}
	return nil, fmt.Errorf("pool not found: %s", name)
}

// GetAllPools returns all registered pools
func (r *PoolRegistry) GetAllPools() []LegacyPool {
	return r.pools
}

// PoolCount returns the number of registered pools
func (r *PoolRegistry) PoolCount() int {
	return len(r.pools)
}
