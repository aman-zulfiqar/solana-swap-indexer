package server

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/time/rate"
)

// RegisterRoutes configures all API routes, middleware, and error handlers
func RegisterRoutes(e *echo.Echo, h *Handlers, cfg ServerConfig) {
	// Set custom error handler for consistent JSON responses
	e.HTTPErrorHandler = NotFoundJSON()

	// Apply global middleware
	e.Use(SetJSONContentType) // Ensure all responses are JSON
	e.Use(SetNoCacheHeaders)  // Prevent caching of API responses

	// Optional API key authentication
	if cfg.APIKey != "" {
		e.Use(middleware.KeyAuthWithConfig(middleware.KeyAuthConfig{
			KeyLookup: "header:X-API-Key", // Look for API key in X-API-Key header
			Validator: func(key string, c echo.Context) (bool, error) {
				return key == cfg.APIKey, nil // Simple string comparison
			},
		}))
	}

	// API v1 routes
	v1 := e.Group("/v1")
	v1.GET("/health", h.Health)            // Health check endpoint
	v1.POST("/echo", h.Echo)               // Echo endpoint for testing
	v1.GET("/swaps/recent", h.RecentSwaps) // Recent swap events
	v1.GET("/prices/:token", h.Price)      // Token price lookup

	// AI endpoints with rate limiting
	aigroup := v1.Group("/ai")
	aigroup.Use(middleware.RateLimiter(middleware.NewRateLimiterMemoryStoreWithConfig(middleware.RateLimiterMemoryStoreConfig{
		Rate:      rate.Limit(0.2), // 1 request every 5 seconds
		Burst:     2,               // Allow burst of 2 requests
		ExpiresIn: 2 * time.Minute, // Rate limit window
	})))
	aigroup.POST("/ask", h.AIAsk) // Natural language to SQL endpoint

	// Feature flags CRUD endpoints
	flagGroup := v1.Group("/flags")
	flagGroup.GET("", h.FlagsList)           // List all flags
	flagGroup.POST("", h.FlagsUpsert)        // Create new flag
	flagGroup.GET("/:key", h.FlagsGet)       // Get specific flag
	flagGroup.PUT("/:key", h.FlagsUpdate)    // Update existing flag
	flagGroup.DELETE("/:key", h.FlagsDelete) // Delete flag

	// Catch-all route for 404 responses
	e.RouteNotFound("/*", func(c echo.Context) error {
		return c.JSON(http.StatusNotFound, ErrorResponse{Error: "not found", Code: http.StatusNotFound})
	})
}
