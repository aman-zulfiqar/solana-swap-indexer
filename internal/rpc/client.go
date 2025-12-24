package rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

// Client is an HTTP client with retry and timeout support for Solana RPC
type Client struct {
	httpClient   *http.Client
	baseURL      string
	maxRetries   int
	retryBackoff time.Duration
	logger       *logrus.Logger
}

// ClientConfig holds configuration for the RPC client
type ClientConfig struct {
	BaseURL      string
	Timeout      time.Duration
	MaxRetries   int
	RetryBackoff time.Duration
	Logger       *logrus.Logger
}

// NewClient creates a new RPC client with retry support
func NewClient(cfg ClientConfig) *Client {
	if cfg.Logger == nil {
		cfg.Logger = logrus.New()
	}

	return &Client{
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		baseURL:      cfg.BaseURL,
		maxRetries:   cfg.MaxRetries,
		retryBackoff: cfg.RetryBackoff,
		logger:       cfg.Logger,
	}
}

// Call makes a JSON-RPC call with retry logic
func (c *Client) Call(ctx context.Context, method string, params interface{}, result interface{}) error {
	body := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  method,
		"params":  params,
	}

	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	var lastErr error
	backoff := c.retryBackoff

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			c.logger.WithFields(logrus.Fields{
				"attempt": attempt,
				"backoff": backoff,
				"method":  method,
			}).Debug("retrying RPC call")

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
			backoff *= 2 // exponential backoff
		}

		resp, err := c.doRequest(ctx, data)
		if err != nil {
			lastErr = err
			continue
		}

		if err := json.Unmarshal(resp, result); err != nil {
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}

		return nil
	}

	return fmt.Errorf("max retries exceeded: %w", lastErr)
}

func (c *Client) doRequest(ctx context.Context, data []byte) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL, bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Handle rate limiting
	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("rate limited (429)")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return body, nil
}

// GetSignaturesForAddress fetches transaction signatures for a program address
func (c *Client) GetSignaturesForAddress(ctx context.Context, address string, opts map[string]interface{}) (*SignaturesResponse, error) {
	params := []interface{}{address, opts}

	var result SignaturesResponse
	if err := c.Call(ctx, "getSignaturesForAddress", params, &result); err != nil {
		return nil, err
	}

	if result.Error != nil {
		return nil, result.Error
	}

	return &result, nil
}

// GetTransaction fetches full transaction details
func (c *Client) GetTransaction(ctx context.Context, signature string) (*TransactionResponse, error) {
	params := []interface{}{
		signature,
		map[string]interface{}{
			"encoding":                       "jsonParsed",
			"maxSupportedTransactionVersion": 0,
		},
	}

	var result TransactionResponse
	if err := c.Call(ctx, "getTransaction", params, &result); err != nil {
		return nil, err
	}

	if result.Error != nil {
		return nil, result.Error
	}

	return &result, nil
}
