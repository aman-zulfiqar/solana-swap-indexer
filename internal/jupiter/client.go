package jupiter

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	BaseURL string
	APIKey  string
	HTTP    *http.Client
}

func NewClient(baseURL, apiKey string) *Client {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		baseURL = "https://api.jup.ag/swap/v1"
	}
	return &Client{
		BaseURL: baseURL,
		APIKey:  strings.TrimSpace(apiKey),
		HTTP: &http.Client{
			Timeout: 12 * time.Second,
		},
	}
}

type HTTPError struct {
	StatusCode int
	Body       []byte
}

func (e *HTTPError) Error() string {
	b := strings.TrimSpace(string(e.Body))
	if b == "" {
		return fmt.Sprintf("jupiter http %d", e.StatusCode)
	}
	return fmt.Sprintf("jupiter http %d: %s", e.StatusCode, b)
}

func (c *Client) Quote(ctx context.Context, req QuoteRequest) (*QuoteResponse, error) {
	if strings.TrimSpace(req.InputMint) == "" {
		return nil, fmt.Errorf("inputMint is required")
	}
	if strings.TrimSpace(req.OutputMint) == "" {
		return nil, fmt.Errorf("outputMint is required")
	}
	if strings.TrimSpace(req.Amount) == "" {
		return nil, fmt.Errorf("amount is required")
	}

	q := url.Values{}
	q.Set("inputMint", req.InputMint)
	q.Set("outputMint", req.OutputMint)
	q.Set("amount", req.Amount)

	if req.SlippageBps != nil {
		q.Set("slippageBps", fmt.Sprintf("%d", *req.SlippageBps))
	}
	if req.SwapMode != "" {
		q.Set("swapMode", req.SwapMode)
	}
	if len(req.Dexes) > 0 {
		q.Set("dexes", strings.Join(req.Dexes, ","))
	}
	if len(req.ExcludeDexes) > 0 {
		q.Set("excludeDexes", strings.Join(req.ExcludeDexes, ","))
	}
	if req.RestrictIntermediateTokens != nil {
		q.Set("restrictIntermediateTokens", fmt.Sprintf("%t", *req.RestrictIntermediateTokens))
	}
	if req.OnlyDirectRoutes != nil {
		q.Set("onlyDirectRoutes", fmt.Sprintf("%t", *req.OnlyDirectRoutes))
	}
	if req.AsLegacyTransaction != nil {
		q.Set("asLegacyTransaction", fmt.Sprintf("%t", *req.AsLegacyTransaction))
	}
	if req.PlatformFeeBps != nil {
		q.Set("platformFeeBps", fmt.Sprintf("%d", *req.PlatformFeeBps))
	}
	if req.MaxAccounts != nil {
		q.Set("maxAccounts", fmt.Sprintf("%d", *req.MaxAccounts))
	}
	if req.InstructionVersion != "" {
		q.Set("instructionVersion", req.InstructionVersion)
	}
	if req.DynamicSlippage != nil {
		q.Set("dynamicSlippage", fmt.Sprintf("%t", *req.DynamicSlippage))
	}

	u := c.BaseURL + "/quote?" + q.Encode()
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("accept", "application/json")
	if c.APIKey != "" {
		httpReq.Header.Set("x-api-key", c.APIKey)
	}

	res, err := c.HTTP.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, _ := io.ReadAll(res.Body)
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, &HTTPError{StatusCode: res.StatusCode, Body: body}
	}

	var out QuoteResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("failed to decode jupiter quote response: %w", err)
	}
	return &out, nil
}
