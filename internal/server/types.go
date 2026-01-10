package server

// ErrorResponse represents a standardized error response format
type ErrorResponse struct {
	Error   string `json:"error"`             // Human-readable error message
	Code    int    `json:"code"`              // HTTP status code
	Details any    `json:"details,omitempty"` // Additional error details (dev mode only)
}

// HealthResponse represents the health check response
type HealthResponse struct {
	OK bool `json:"ok"` // Service health status
}

// SwapsRecentResponse represents recent swaps response (deprecated - use inline struct)
type SwapsRecentResponse struct {
	Items any `json:"items"` // List of swap events
}

// PriceResponse represents token price information
type PriceResponse struct {
	Token string  `json:"token"` // Token symbol (uppercase)
	Price float64 `json:"price"` // Current price
}

// FlagUpsertRequest represents a request to create or update a feature flag
type FlagUpsertRequest struct {
	Key   string `json:"key"`   // Flag key (must match regex pattern)
	Value bool   `json:"value"` // Flag value (true/false)
}

// FlagUpdateRequest represents a request to update an existing feature flag
type FlagUpdateRequest struct {
	Value bool `json:"value"` // New flag value
}

// AIAskRequest represents a natural language query request
type AIAskRequest struct {
	Question string `json:"question"` // Natural language question about swap data
	Model    string `json:"model"`    // Optional AI model override
}

// AIAskResponse represents the response from an AI query
type AIAskResponse struct {
	SQL    string `json:"sql"`     // Generated SQL query
	Answer string `json:"answer"`  // Natural language answer
	TookMs int64  `json:"took_ms"` // Execution time in milliseconds
}
