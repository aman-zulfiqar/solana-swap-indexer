package rpc

// RPCError represents a JSON-RPC error response
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *RPCError) Error() string {
	return e.Message
}

// SignatureInfo represents a transaction signature from getSignaturesForAddress
type SignatureInfo struct {
	Signature string      `json:"signature"`
	Slot      int64       `json:"slot"`
	Err       interface{} `json:"err"`
	BlockTime int64       `json:"blockTime"`
}

// SignaturesResponse is the response from getSignaturesForAddress
type SignaturesResponse struct {
	Result []SignatureInfo `json:"result"`
	Error  *RPCError       `json:"error"`
}

// TokenAmount represents token balance information
type TokenAmount struct {
	Amount         string  `json:"amount"`
	Decimals       int     `json:"decimals"`
	UIAmountString string  `json:"uiAmountString"`
	UIAmount       float64 `json:"uiAmount"`
}

// TokenBalance represents a token balance entry
type TokenBalance struct {
	AccountIndex  int         `json:"accountIndex"`
	Mint          string      `json:"mint"`
	UITokenAmount TokenAmount `json:"uiTokenAmount"`
}

// TransactionMeta contains metadata about a transaction
type TransactionMeta struct {
	Err               interface{}    `json:"err"`
	PreBalances       []int64        `json:"preBalances"`
	PostBalances      []int64        `json:"postBalances"`
	PreTokenBalances  []TokenBalance `json:"preTokenBalances"`
	PostTokenBalances []TokenBalance `json:"postTokenBalances"`
}

// AccountKey represents an account in a transaction
type AccountKey struct {
	Pubkey string `json:"pubkey"`
}

// TransactionMessage contains the transaction message
type TransactionMessage struct {
	AccountKeys []AccountKey `json:"accountKeys"`
}

// Transaction represents a parsed transaction
type Transaction struct {
	Message TransactionMessage `json:"message"`
}

// TransactionResult contains the full transaction data
type TransactionResult struct {
	Meta        *TransactionMeta `json:"meta"`
	Transaction *Transaction     `json:"transaction"`
}

// TransactionResponse is the response from getTransaction
type TransactionResponse struct {
	Result *TransactionResult `json:"result"`
	Error  *RPCError          `json:"error"`
}

// BalanceChange represents a token balance change in a swap
type BalanceChange struct {
	Mint   string
	Amount float64
}
