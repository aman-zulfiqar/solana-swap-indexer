package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/aman-zulfiqar/solana-swap-indexer/internal/ai"
	"github.com/aman-zulfiqar/solana-swap-indexer/internal/flags"
	"github.com/aman-zulfiqar/solana-swap-indexer/internal/jupiter"
	"github.com/aman-zulfiqar/solana-swap-indexer/internal/storage"
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

// Handlers contains all dependencies for API endpoint handlers
type Handlers struct {
	Cache        storage.SwapCache // Redis-backed swap data cache
	Flags        *flags.Store      // Redis-backed feature flags store
	AI           *ai.Agent         // AI agent for natural language queries
	AIBaseConfig ai.AgentConfig    // Base configuration for AI agents
	DevMode      bool              // Enable detailed error responses in development
	Logger       *logrus.Logger    // Structured logger
	Jupiter      *jupiter.Client   // Jupiter Quote API client (optional)
}

// err returns a standardized JSON error response
// In dev mode, includes additional error details for debugging
func (h *Handlers) err(c echo.Context, code int, msg string, details any) error {
	resp := ErrorResponse{Error: msg, Code: code}
	if h.DevMode && details != nil {
		resp.Details = details
	}
	return c.JSON(code, resp)
}

// withTimeout creates a context with timeout, defaulting to 10 seconds if duration <= 0
func (h *Handlers) withTimeout(ctx context.Context, d time.Duration) (context.Context, context.CancelFunc) {
	if d <= 0 {
		d = 10 * time.Second
	}
	return context.WithTimeout(ctx, d)
}

// Health returns a simple health check endpoint
func (h *Handlers) Health(c echo.Context) error {
	return c.JSON(http.StatusOK, HealthResponse{OK: true})
}

// Echo returns the received JSON payload as-is (useful for testing)
func (h *Handlers) Echo(c echo.Context) error {
	var v any
	dec := json.NewDecoder(c.Request().Body)
	if err := dec.Decode(&v); err != nil {
		return h.err(c, http.StatusBadRequest, "invalid json", nil)
	}
	return c.JSON(http.StatusOK, v)
}

// RecentSwaps returns the most recent swap events with optional limit parameter
// Accepts limit query parameter (default: 100, range: 1-200)
func (h *Handlers) RecentSwaps(c echo.Context) error {
	limitStr := c.QueryParam("limit")
	limit := 100
	if limitStr != "" {
		n, err := strconv.Atoi(limitStr)
		if err != nil {
			return h.err(c, http.StatusBadRequest, "invalid limit", map[string]any{"limit": "must be an integer"})
		}
		limit = n
	}
	if limit < 1 || limit > 200 {
		return h.err(c, http.StatusBadRequest, "invalid limit", map[string]any{"limit": "min 1 max 200"})
	}

	ctx, cancel := h.withTimeout(c.Request().Context(), 5*time.Second)
	defer cancel()

	items, err := h.Cache.GetRecentSwaps(ctx, int64(limit))
	if err != nil {
		return h.err(c, http.StatusInternalServerError, "failed to get swaps", nil)
	}
	return c.JSON(http.StatusOK, map[string]any{"items": items})
}

// Price returns the current price for a given token symbol
// Token parameter is case-insensitive and will be normalized to uppercase
func (h *Handlers) Price(c echo.Context) error {
	token := strings.TrimSpace(c.Param("token"))
	if token == "" {
		return h.err(c, http.StatusBadRequest, "invalid token", nil)
	}
	token = strings.ToUpper(token)

	ctx, cancel := h.withTimeout(c.Request().Context(), 3*time.Second)
	defer cancel()

	price, err := h.Cache.GetPrice(ctx, token)
	if err != nil {
		return h.err(c, http.StatusInternalServerError, "failed to get price", nil)
	}
	return c.JSON(http.StatusOK, PriceResponse{Token: token, Price: price})
}

// FlagsUpsert creates or updates a feature flag with the given key and value
// Validates key format and returns the created/updated flag
func (h *Handlers) FlagsUpsert(c echo.Context) error {
	var req FlagUpsertRequest
	if err := c.Bind(&req); err != nil {
		return h.err(c, http.StatusBadRequest, "invalid json", nil)
	}
	if err := flags.ValidateKey(req.Key); err != nil {
		return h.err(c, http.StatusBadRequest, "invalid key", map[string]any{"key": "invalid format"})
	}

	ctx, cancel := h.withTimeout(c.Request().Context(), 3*time.Second)
	defer cancel()

	out, err := h.Flags.Upsert(ctx, req.Key, req.Value)
	if err != nil {
		return h.err(c, http.StatusInternalServerError, "failed to upsert flag", nil)
	}
	return c.JSON(http.StatusOK, out)
}

// FlagsUpdate updates an existing feature flag with the given key
// Validates key format and returns the updated flag
func (h *Handlers) FlagsUpdate(c echo.Context) error {
	key := c.Param("key")
	if err := flags.ValidateKey(key); err != nil {
		return h.err(c, http.StatusBadRequest, "invalid key", map[string]any{"key": "invalid format"})
	}
	var req FlagUpdateRequest
	if err := c.Bind(&req); err != nil {
		return h.err(c, http.StatusBadRequest, "invalid json", nil)
	}

	ctx, cancel := h.withTimeout(c.Request().Context(), 3*time.Second)
	defer cancel()

	out, err := h.Flags.Upsert(ctx, key, req.Value)
	if err != nil {
		return h.err(c, http.StatusInternalServerError, "failed to update flag", nil)
	}
	return c.JSON(http.StatusOK, out)
}

// FlagsGet retrieves a feature flag by its key
// Returns 404 if flag doesn't exist
func (h *Handlers) FlagsGet(c echo.Context) error {
	key := c.Param("key")
	if err := flags.ValidateKey(key); err != nil {
		return h.err(c, http.StatusBadRequest, "invalid key", map[string]any{"key": "invalid format"})
	}

	ctx, cancel := h.withTimeout(c.Request().Context(), 3*time.Second)
	defer cancel()

	out, err := h.Flags.Get(ctx, key)
	if err != nil {
		if errors.Is(err, flags.ErrNotFound) {
			return h.err(c, http.StatusNotFound, "flag not found", nil)
		}
		return h.err(c, http.StatusInternalServerError, "failed to get flag", nil)
	}
	return c.JSON(http.StatusOK, out)
}

// FlagsList returns all feature flags in the system
func (h *Handlers) FlagsList(c echo.Context) error {
	ctx, cancel := h.withTimeout(c.Request().Context(), 5*time.Second)
	defer cancel()

	items, err := h.Flags.List(ctx)
	if err != nil {
		return h.err(c, http.StatusInternalServerError, "failed to list flags", nil)
	}
	return c.JSON(http.StatusOK, map[string]any{"items": items})
}

// FlagsDelete removes a feature flag by its key
// Returns 204 No Content on successful deletion
func (h *Handlers) FlagsDelete(c echo.Context) error {
	key := c.Param("key")
	if err := flags.ValidateKey(key); err != nil {
		return h.err(c, http.StatusBadRequest, "invalid key", map[string]any{"key": "invalid format"})
	}

	ctx, cancel := h.withTimeout(c.Request().Context(), 3*time.Second)
	defer cancel()

	if err := h.Flags.Delete(ctx, key); err != nil {
		return h.err(c, http.StatusInternalServerError, "failed to delete flag", nil)
	}
	return c.NoContent(http.StatusNoContent)
}

// AIAsk processes natural language questions about swap data using AI
// Supports optional model override for one-off requests
// Returns SQL query and answer with execution time
func (h *Handlers) AIAsk(c echo.Context) error {
	if h.AI == nil {
		return h.err(c, http.StatusBadRequest, "ai is not configured", nil)
	}

	var req AIAskRequest
	if err := c.Bind(&req); err != nil {
		return h.err(c, http.StatusBadRequest, "invalid json", nil)
	}
	req.Question = strings.TrimSpace(req.Question)
	if req.Question == "" {
		return h.err(c, http.StatusBadRequest, "question is required", map[string]any{"question": "required"})
	}

	ctx, cancel := h.withTimeout(c.Request().Context(), 45*time.Second)
	defer cancel()

	start := time.Now()

	// Use default AI agent or create temporary one with custom model
	agent := h.AI
	var tmp *ai.Agent
	if m := strings.TrimSpace(req.Model); m != "" {
		cfg := h.AIBaseConfig
		cfg.Model = m
		a, err := ai.NewAgent(ctx, cfg)
		if err != nil {
			return h.err(c, http.StatusInternalServerError, "failed to create ai agent", nil)
		}
		tmp = a
		agent = a
		defer func() {
			_ = tmp.Close() // Clean up temporary agent
		}()
	}

	res, err := agent.Ask(ctx, req.Question)
	if err != nil {
		return h.err(c, http.StatusInternalServerError, "ai ask failed", map[string]any{"err": err.Error()})
	}

	return c.JSON(http.StatusOK, AIAskResponse{SQL: res.SQL, Answer: res.Answer, TookMs: time.Since(start).Milliseconds()})
}
