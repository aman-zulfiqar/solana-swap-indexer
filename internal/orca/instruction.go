package orca

import (
	"encoding/binary"
	"fmt"

	"github.com/gagliardetto/solana-go"
)

// BuildLegacySwapInstruction constructs an SPL Token Swap style instruction
// for Orca legacy pools
func BuildLegacySwapInstruction(
	pool *LegacyPool,
	amountIn uint64,
	minAmountOut uint64,
	userAuthority solana.PublicKey, // The signer (user's wallet)
	userTokenAccountIn solana.PublicKey, // User's source token account
	userTokenAccountOut solana.PublicKey, // User's destination token account
	aToB bool, // true = swap A->B, false = swap B->A
) (solana.Instruction, error) {

	if pool == nil {
		return nil, fmt.Errorf("pool cannot be nil")
	}

	// Determine vault direction
	poolSource := pool.VaultA
	poolDest := pool.VaultB
	if !aToB {
		poolSource = pool.VaultB
		poolDest = pool.VaultA
	}

	// SPL Token Swap instruction account order:
	// 0. swap_state (the pool/swap account)
	// 1. authority (PDA that controls vaults)
	// 2. user_transfer_authority (signer)
	// 3. user_source (user's input token account)
	// 4. pool_source (vault being swapped from)
	// 5. pool_destination (vault being swapped to)
	// 6. user_destination (user's output token account)
	// 7. pool_mint (LP token mint)
	// 8. fee_account (where fees go)
	// 9. token_program
	// 10. host_fee_account (optional)

	accounts := []*solana.AccountMeta{
		{PublicKey: pool.SwapAccount, IsWritable: true, IsSigner: false},
		{PublicKey: pool.Authority, IsWritable: false, IsSigner: false},
		{PublicKey: userAuthority, IsWritable: false, IsSigner: true},
		{PublicKey: userTokenAccountIn, IsWritable: true, IsSigner: false},
		{PublicKey: poolSource, IsWritable: true, IsSigner: false},
		{PublicKey: poolDest, IsWritable: true, IsSigner: false},
		{PublicKey: userTokenAccountOut, IsWritable: true, IsSigner: false},
		{PublicKey: pool.PoolMint, IsWritable: true, IsSigner: false},
		{PublicKey: pool.FeeAccount, IsWritable: true, IsSigner: false},
		{PublicKey: solana.TokenProgramID, IsWritable: false, IsSigner: false},
	}

	// Add optional host fee account
	if pool.HostFeeAccount != nil {
		accounts = append(accounts, &solana.AccountMeta{
			PublicKey:  *pool.HostFeeAccount,
			IsWritable: true,
			IsSigner:   false,
		})
	}

	// Instruction data layout for SPL Token Swap:
	// [0] = instruction discriminator (1 = Swap)
	// [1:9] = amount_in (u64, little-endian)
	// [9:17] = minimum_amount_out (u64, little-endian)
	data := make([]byte, 17)
	data[0] = 1 // Swap instruction discriminator
	binary.LittleEndian.PutUint64(data[1:9], amountIn)
	binary.LittleEndian.PutUint64(data[9:17], minAmountOut)

	return solana.NewInstruction(
		pool.ProgramID,
		accounts,
		data,
	), nil
}

// DetermineSwapDirection determines if swap is A->B based on input mint
func DetermineSwapDirection(pool *LegacyPool, inputMint solana.PublicKey) (bool, error) {
	if pool.TokenMintA.Equals(inputMint) {
		return true, nil // A -> B
	}
	if pool.TokenMintB.Equals(inputMint) {
		return false, nil // B -> A
	}
	return false, fmt.Errorf("input mint %s does not match pool mints", inputMint)
}
