package swapengine

import (
	"encoding/binary"
	"fmt"

	"github.com/gagliardetto/solana-go"
)

var (
	// SPL Associated Token Account program
	associatedTokenProgramID = solana.MustPublicKeyFromBase58("ATokenGPvbdGVxr1b2hvZbsiqW5xWH25efTNsLJA8knL")
)

// FindAssociatedTokenAddress derives the ATA PDA for (owner, mint).
func FindAssociatedTokenAddress(owner, mint solana.PublicKey) (ata solana.PublicKey, bump uint8, err error) {
	// Seeds: [owner, token_program, mint]
	return solana.FindProgramAddress(
		[][]byte{
			owner.Bytes(),
			solana.TokenProgramID.Bytes(),
			mint.Bytes(),
		},
		associatedTokenProgramID,
	)
}

// NewCreateAssociatedTokenAccountIx builds an instruction to create an ATA.
// Account order (ATA program):
// 0. payer (signer, writable)
// 1. ata (writable)
// 2. owner (read-only)
// 3. mint (read-only)
// 4. system_program
// 5. token_program
// 6. rent_sysvar
func NewCreateAssociatedTokenAccountIx(
	payer solana.PublicKey,
	ata solana.PublicKey,
	owner solana.PublicKey,
	mint solana.PublicKey,
) solana.Instruction {
	accounts := []*solana.AccountMeta{
		{PublicKey: payer, IsSigner: true, IsWritable: true},
		{PublicKey: ata, IsSigner: false, IsWritable: true},
		{PublicKey: owner, IsSigner: false, IsWritable: false},
		{PublicKey: mint, IsSigner: false, IsWritable: false},
		{PublicKey: solana.SystemProgramID, IsSigner: false, IsWritable: false},
		{PublicKey: solana.TokenProgramID, IsSigner: false, IsWritable: false},
		{PublicKey: solana.SysVarRentPubkey, IsSigner: false, IsWritable: false},
	}

	// ATA create instruction data is empty.
	return solana.NewInstruction(associatedTokenProgramID, accounts, nil)
}

// NewSystemTransferIx builds a SystemProgram transfer instruction.
func NewSystemTransferIx(from, to solana.PublicKey, lamports uint64) solana.Instruction {
	// SystemProgram instruction layout:
	// u32: instruction index (2 = Transfer)
	// u64: lamports
	data := make([]byte, 4+8)
	binary.LittleEndian.PutUint32(data[0:4], 2)
	binary.LittleEndian.PutUint64(data[4:12], lamports)

	accounts := []*solana.AccountMeta{
		{PublicKey: from, IsSigner: true, IsWritable: true},
		{PublicKey: to, IsSigner: false, IsWritable: true},
	}
	return solana.NewInstruction(solana.SystemProgramID, accounts, data)
}

// NewTokenSyncNativeIx builds a SPL Token SyncNative instruction.
func NewTokenSyncNativeIx(nativeAccount solana.PublicKey) solana.Instruction {
	// TokenProgram instruction index 17 = SyncNative
	data := []byte{17}
	accounts := []*solana.AccountMeta{
		{PublicKey: nativeAccount, IsSigner: false, IsWritable: true},
	}
	return solana.NewInstruction(solana.TokenProgramID, accounts, data)
}

// NewTokenCloseAccountIx builds a SPL Token CloseAccount instruction.
func NewTokenCloseAccountIx(account, destination, owner solana.PublicKey) solana.Instruction {
	// TokenProgram instruction index 9 = CloseAccount
	data := []byte{9}
	accounts := []*solana.AccountMeta{
		{PublicKey: account, IsSigner: false, IsWritable: true},
		{PublicKey: destination, IsSigner: false, IsWritable: true},
		{PublicKey: owner, IsSigner: true, IsWritable: false},
	}
	return solana.NewInstruction(solana.TokenProgramID, accounts, data)
}

func requirePubkey(pk solana.PublicKey, name string) error {
	if pk.IsZero() {
		return fmt.Errorf("%s is zero", name)
	}
	return nil
}

