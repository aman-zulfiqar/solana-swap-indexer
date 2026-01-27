package jupiter

type QuoteRequest struct {
	InputMint  string
	OutputMint string
	Amount     string // raw integer as string (uint64)

	SlippageBps *uint16
	SwapMode    string // ExactIn | ExactOut

	Dexes        []string
	ExcludeDexes []string

	RestrictIntermediateTokens *bool
	OnlyDirectRoutes           *bool
	AsLegacyTransaction        *bool

	PlatformFeeBps *uint16
	MaxAccounts    *uint64

	InstructionVersion string // V1 | V2
	DynamicSlippage    *bool
}

type QuoteResponse struct {
	InputMint            string          `json:"inputMint"`
	OutputMint           string          `json:"outputMint"`
	InAmount             string          `json:"inAmount"`
	OutAmount            string          `json:"outAmount"`
	OtherAmountThreshold string          `json:"otherAmountThreshold"`
	SwapMode             string          `json:"swapMode"`
	SlippageBps          uint16          `json:"slippageBps"`
	PlatformFee          *PlatformFee    `json:"platformFee,omitempty"`
	PriceImpactPct       string          `json:"priceImpactPct"`
	RoutePlan            []RoutePlanStep `json:"routePlan"`

	ContextSlot uint64  `json:"contextSlot,omitempty"`
	TimeTaken   float64 `json:"timeTaken,omitempty"`
}

type PlatformFee struct {
	Amount string `json:"amount,omitempty"`
	FeeBps uint16 `json:"feeBps,omitempty"`
}

type RoutePlanStep struct {
	SwapInfo SwapInfo `json:"swapInfo"`
	Percent  *uint8   `json:"percent,omitempty"`
	Bps      uint16   `json:"bps"`
}

type SwapInfo struct {
	AmmKey     string `json:"ammKey"`
	Label      string `json:"label,omitempty"`
	InputMint  string `json:"inputMint"`
	OutputMint string `json:"outputMint"`
	InAmount   string `json:"inAmount"`
	OutAmount  string `json:"outAmount"`

	FeeAmount *string `json:"feeAmount,omitempty"`
	FeeMint   *string `json:"feeMint,omitempty"`
}
