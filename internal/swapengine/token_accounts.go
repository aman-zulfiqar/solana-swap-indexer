package swapengine

import (
	"context"
	"fmt"

	"github.com/aman-zulfiqar/solana-swap-indexer/internal/wallet"
	"github.com/gagliardetto/solana-go"
)

// ResolvedTokenAccount describes a token account to use for a swap plus any
// instructions needed to make it usable (e.g. create ATA).
type ResolvedTokenAccount struct {
	Account solana.PublicKey
	Created bool // true if this resolver will create the account in PreIxs
	PreIxs  []solana.Instruction
}

// DefaultTokenAccountResolver resolves the owner's ATA for a given mint.
// For wSOL, it returns the ATA as well (wrapping/unwrapping is handled by Executor).
type DefaultTokenAccountResolver struct {
	w *wallet.Wallet
}

func NewDefaultTokenAccountResolver(w *wallet.Wallet) *DefaultTokenAccountResolver {
	return &DefaultTokenAccountResolver{w: w}
}

func (r *DefaultTokenAccountResolver) Resolve(ctx context.Context, owner solana.PublicKey, mint solana.PublicKey) (*ResolvedTokenAccount, error) {
	if r == nil || r.w == nil {
		return nil, fmt.Errorf("token account resolver: wallet is nil")
	}

	ata, _, err := FindAssociatedTokenAddress(owner, mint)
	if err != nil {
		return nil, err
	}

	exists, err := r.w.AccountExists(ctx, ata)
	if err != nil {
		return nil, err
	}
	if exists {
		return &ResolvedTokenAccount{Account: ata, Created: false}, nil
	}

	// Create ATA (payer=owner).
	createATA := NewCreateAssociatedTokenAccountIx(owner, ata, owner, mint)
	return &ResolvedTokenAccount{
		Account: ata,
		Created: true,
		PreIxs: []solana.Instruction{createATA},
	}, nil
}

