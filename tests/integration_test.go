package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/aman-zulfiqar/solana-swap-indexer/internal/ai"
	"github.com/aman-zulfiqar/solana-swap-indexer/internal/cache"
	"github.com/aman-zulfiqar/solana-swap-indexer/internal/config"
	"github.com/aman-zulfiqar/solana-swap-indexer/internal/flags"
	"github.com/aman-zulfiqar/solana-swap-indexer/internal/models"
	"github.com/aman-zulfiqar/solana-swap-indexer/internal/server"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testAPIAddr = ":8091"
	testAPIKey  = "test-api-key-integration"
)

func setupIntegrationTest(t *testing.T) (*server.Server, *redis.Client, func()) {
	// Check if Redis is available
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr: redisAddr,
		DB:   2, // Use different DB for integration tests
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis not available for integration tests: %v", err)
	}

	// Clear test DB
	_ = redisClient.FlushDB(ctx).Err()

	// Create test configuration
	cfg := &config.Config{
		APIAddr: testAPIAddr,
		APIKey:  testAPIKey,
		DevMode: true,
	}

	// Initialize cache and flags store
	logger := logrus.New()
	swapCache := cache.NewRedisCacheFromClient(redisClient, logger)
	flagStore, err := flags.NewStore(redisClient)
	require.NoError(t, err)

	// Create server dependencies
	handlers := &server.Handlers{
		Cache:        swapCache,
		Flags:        flagStore,
		AI:           nil,
		AIBaseConfig: ai.AgentConfig{},
		DevMode:      true,
		Logger:       logger,
	}

	serverConfig := server.ServerConfig{
		Addr:    cfg.APIAddr,
		DevMode: cfg.DevMode,
		APIKey:  cfg.APIKey,
	}

	deps := server.ServerDeps{
		Handlers: handlers,
		Config:   serverConfig,
	}

	srv, err := server.NewServer(deps)
	require.NoError(t, err)

	// Start server in background
	go func() {
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			t.Logf("Server error: %v", err)
		}
	}()

	// Wait for server to be ready
	time.Sleep(100 * time.Millisecond)

	// Cleanup function
	cleanup := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_ = srv.Shutdown(ctx)
		_ = redisClient.FlushDB(ctx).Err()
		_ = redisClient.Close()
	}

	return srv, redisClient, cleanup
}

func makeRequest(t *testing.T, method, url string, body interface{}, expectedStatus int) *http.Response {
	var reqBody *bytes.Buffer
	if body != nil {
		jsonBody, err := json.Marshal(body)
		require.NoError(t, err)
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequest(method, url, reqBody)
	require.NoError(t, err)

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", testAPIKey)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)

	assert.Equal(t, expectedStatus, resp.StatusCode, "Expected status %d, got %d", expectedStatus, resp.StatusCode)

	return resp
}

func TestIntegration_Health(t *testing.T) {
	_, _, cleanup := setupIntegrationTest(t)
	defer cleanup()

	resp := makeRequest(t, http.MethodGet, "http://localhost:8091/v1/health", nil, http.StatusOK)
	defer resp.Body.Close()

	var response server.HealthResponse
	err := json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)

	assert.True(t, response.OK)
}

func TestIntegration_Echo(t *testing.T) {
	_, _, cleanup := setupIntegrationTest(t)
	defer cleanup()

	payload := map[string]interface{}{"message": "hello", "count": 42}
	resp := makeRequest(t, http.MethodPost, "http://localhost:8091/v1/echo", payload, http.StatusOK)
	defer resp.Body.Close()

	var response map[string]interface{}
	err := json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, payload["message"], response["message"])
	assert.Equal(t, payload["count"], response["count"])
}

func TestIntegration_FlagsCRUD(t *testing.T) {
	_, _, cleanup := setupIntegrationTest(t)
	defer cleanup()

	// Create flag
	upsertPayload := map[string]interface{}{"key": "test.flag", "value": true}
	resp := makeRequest(t, http.MethodPost, "http://localhost:8091/v1/flags", upsertPayload, http.StatusOK)
	defer resp.Body.Close()

	var upsertResponse flags.Flag
	err := json.NewDecoder(resp.Body).Decode(&upsertResponse)
	require.NoError(t, err)
	assert.Equal(t, "test.flag", upsertResponse.Key)
	assert.True(t, upsertResponse.Value)
	assert.NotZero(t, upsertResponse.UpdatedAt)

	// Get flag
	resp = makeRequest(t, http.MethodGet, "http://localhost:8091/v1/flags/test.flag", nil, http.StatusOK)
	defer resp.Body.Close()

	var getResponse flags.Flag
	err = json.NewDecoder(resp.Body).Decode(&getResponse)
	require.NoError(t, err)
	assert.Equal(t, "test.flag", getResponse.Key)
	assert.True(t, getResponse.Value)

	// Update flag
	updatePayload := map[string]interface{}{"value": false}
	resp = makeRequest(t, http.MethodPut, "http://localhost:8091/v1/flags/test.flag", updatePayload, http.StatusOK)
	defer resp.Body.Close()

	var updateResponse flags.Flag
	err = json.NewDecoder(resp.Body).Decode(&updateResponse)
	require.NoError(t, err)
	assert.Equal(t, "test.flag", updateResponse.Key)
	assert.False(t, updateResponse.Value)

	// List flags
	resp = makeRequest(t, http.MethodGet, "http://localhost:8091/v1/flags", nil, http.StatusOK)
	defer resp.Body.Close()

	var listResponse struct {
		Items []*flags.Flag `json:"items"`
	}
	err = json.NewDecoder(resp.Body).Decode(&listResponse)
	require.NoError(t, err)
	assert.Len(t, listResponse.Items, 1)
	assert.Equal(t, "test.flag", listResponse.Items[0].Key)
	assert.False(t, listResponse.Items[0].Value)

	// Delete flag
	resp = makeRequest(t, http.MethodDelete, "http://localhost:8091/v1/flags/test.flag", nil, http.StatusNoContent)
	defer resp.Body.Close()

	// Verify deletion
	resp = makeRequest(t, http.MethodGet, "http://localhost:8091/v1/flags/test.flag", nil, http.StatusNotFound)
	defer resp.Body.Close()
}

func TestIntegration_FlagsValidation(t *testing.T) {
	_, _, cleanup := setupIntegrationTest(t)
	defer cleanup()

	// Test invalid key (empty key will fail regex validation)
	invalidPayload := map[string]interface{}{"key": "", "value": true}
	resp := makeRequest(t, http.MethodPost, "http://localhost:8091/v1/flags", invalidPayload, http.StatusBadRequest)
	defer resp.Body.Close()

	var errorResponse server.ErrorResponse
	err := json.NewDecoder(resp.Body).Decode(&errorResponse)
	require.NoError(t, err)
	assert.Contains(t, errorResponse.Error, "invalid key")

	// Test key with invalid characters
	invalidPayload2 := map[string]interface{}{"key": "invalid:key", "value": true}
	resp = makeRequest(t, http.MethodPost, "http://localhost:8091/v1/flags", invalidPayload2, http.StatusBadRequest)
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&errorResponse)
	require.NoError(t, err)
	assert.Contains(t, errorResponse.Error, "invalid flag key")
}

func TestIntegration_SwapsAndPrices(t *testing.T) {
	_, redisClient, cleanup := setupIntegrationTest(t)
	defer cleanup()

	ctx := context.Background()

	// Add some test data to Redis
	swapData := `{"signature":"test_sig","pair":"SOL/USDC","amount_in":1.0,"amount_out":100.0,"price":100.0,"token_in":"SOL","token_out":"USDC"}`
	err := redisClient.LPush(ctx, "swaps:recent", swapData).Err()
	require.NoError(t, err)

	err = redisClient.Set(ctx, "price:SOL", "150.5", 0).Err()
	require.NoError(t, err)

	// Test recent swaps
	resp := makeRequest(t, http.MethodGet, "http://localhost:8091/v1/swaps/recent?limit=5", nil, http.StatusOK)
	defer resp.Body.Close()

	var swapsResponse struct {
		Items []*models.SwapEvent `json:"items"`
	}
	err = json.NewDecoder(resp.Body).Decode(&swapsResponse)
	require.NoError(t, err)
	assert.Len(t, swapsResponse.Items, 1)
	assert.Equal(t, "test_sig", swapsResponse.Items[0].Signature)

	// Test price
	resp = makeRequest(t, http.MethodGet, "http://localhost:8091/v1/prices/SOL", nil, http.StatusOK)
	defer resp.Body.Close()

	var priceResponse server.PriceResponse
	err = json.NewDecoder(resp.Body).Decode(&priceResponse)
	require.NoError(t, err)
	assert.Equal(t, "SOL", priceResponse.Token)
	assert.Equal(t, 150.5, priceResponse.Price)

	// Test unknown token price (should return 0)
	resp = makeRequest(t, http.MethodGet, "http://localhost:8091/v1/prices/UNKNOWN", nil, http.StatusOK)
	defer resp.Body.Close()

	var unknownPriceResponse server.PriceResponse
	err = json.NewDecoder(resp.Body).Decode(&unknownPriceResponse)
	require.NoError(t, err)
	assert.Equal(t, "UNKNOWN", unknownPriceResponse.Token)
	assert.Equal(t, 0.0, unknownPriceResponse.Price)
}

func TestIntegration_SwapsValidation(t *testing.T) {
	_, _, cleanup := setupIntegrationTest(t)
	defer cleanup()

	// Test invalid limit
	resp := makeRequest(t, http.MethodGet, "http://localhost:8091/v1/swaps/recent?limit=500", nil, http.StatusBadRequest)
	defer resp.Body.Close()

	var errorResponse server.ErrorResponse
	err := json.NewDecoder(resp.Body).Decode(&errorResponse)
	require.NoError(t, err)
	assert.Contains(t, errorResponse.Error, "invalid limit")
}

func TestIntegration_Authentication(t *testing.T) {
	_, _, cleanup := setupIntegrationTest(t)
	defer cleanup()

	// Test without API key
	req, err := http.NewRequest(http.MethodGet, "http://localhost:8091/v1/health", nil)
	require.NoError(t, err)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	// Test with invalid API key
	req, err = http.NewRequest(http.MethodGet, "http://localhost:8091/v1/health", nil)
	require.NoError(t, err)
	req.Header.Set("X-API-Key", "invalid-key")

	resp, err = client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestIntegration_ErrorHandling(t *testing.T) {
	_, _, cleanup := setupIntegrationTest(t)
	defer cleanup()

	// Test 404 for non-existent endpoint
	req, err := http.NewRequest(http.MethodGet, "http://localhost:8091/v1/nonexistent", nil)
	require.NoError(t, err)
	req.Header.Set("X-API-Key", testAPIKey)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	var errorResponse server.ErrorResponse
	err = json.NewDecoder(resp.Body).Decode(&errorResponse)
	require.NoError(t, err)
	assert.Equal(t, "not found", errorResponse.Error)
	assert.Equal(t, http.StatusNotFound, errorResponse.Code)

	// Test invalid JSON
	req, err = http.NewRequest(http.MethodPost, "http://localhost:8091/v1/echo", bytes.NewReader([]byte("invalid json")))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", testAPIKey)

	resp, err = client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	err = json.NewDecoder(resp.Body).Decode(&errorResponse)
	require.NoError(t, err)
	assert.Contains(t, errorResponse.Error, "invalid JSON")
}

func TestIntegration_ConcurrentRequests(t *testing.T) {
	_, _, cleanup := setupIntegrationTest(t)
	defer cleanup()

	const numRequests = 50
	const numGoroutines = 10

	results := make(chan error, numRequests)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			for j := 0; j < numRequests/numGoroutines; j++ {
				resp := makeRequest(t, http.MethodGet, "http://localhost:8091/v1/health", nil, http.StatusOK)
				resp.Body.Close()
				results <- nil
			}
		}()
	}

	// Collect all results
	for i := 0; i < numRequests; i++ {
		err := <-results
		assert.NoError(t, err)
	}
}

func TestIntegration_RateLimiting(t *testing.T) {
	_, _, cleanup := setupIntegrationTest(t)
	defer cleanup()

	// Note: This is a basic test. In a real scenario, you'd want to test
	// the actual rate limiting behavior more thoroughly.

	// Make multiple requests quickly to test rate limiting
	for i := 0; i < 5; i++ {
		resp := makeRequest(t, http.MethodGet, "http://localhost:8091/v1/health", nil, http.StatusOK)
		resp.Body.Close()
	}

	// If we get here without rate limiting errors, the basic functionality works
	// In a more comprehensive test, you'd verify the rate limiting headers
	// and behavior when limits are exceeded
}
