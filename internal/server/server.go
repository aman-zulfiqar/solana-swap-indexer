package server

import (
	"context"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// ServerConfig holds configuration for the HTTP server
type ServerConfig struct {
	Addr    string // Server bind address (e.g., ":8090")
	DevMode bool   // Enable development mode (detailed error responses)
	APIKey  string // Optional API key for authentication
}

// ServerDeps contains dependencies required to create a new Server
type ServerDeps struct {
	Handlers *Handlers
	Config   ServerConfig
}

// Server wraps Echo HTTP server with additional lifecycle management
type Server struct {
	e      *echo.Echo
	cfg    ServerConfig
	closed chan struct{} // Channel to signal server shutdown completion
}

// NewServer creates a new HTTP server with the given dependencies
func NewServer(deps ServerDeps) (*Server, error) {
	e := echo.New()
	// Suppress startup banner and port logging for cleaner output
	e.HideBanner = true
	e.HidePort = true

	// Add standard middleware for recovery and request logging
	e.Use(middleware.Recover())
	e.Use(middleware.Logger())

	// Configure server timeouts for robustness
	e.Server.ReadTimeout = 15 * time.Second  // Max time to read request headers
	e.Server.WriteTimeout = 75 * time.Second // Max time to write response
	e.Server.IdleTimeout = 60 * time.Second  // Max time to wait for next request

	h := deps.Handlers
	RegisterRoutes(e, h, deps.Config)

	return &Server{e: e, cfg: deps.Config, closed: make(chan struct{})}, nil
}

// Start begins serving HTTP requests on the configured address
func (s *Server) Start() error {
	return s.e.Start(s.cfg.Addr)
}

// Shutdown gracefully shuts down the server with a 10-second timeout
func (s *Server) Shutdown(ctx context.Context) error {
	defer close(s.closed) // Signal that shutdown is complete
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return s.e.Shutdown(ctx)
}

// WaitClosed blocks until the server is fully shut down or context times out
func (s *Server) WaitClosed(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-s.closed:
		return nil
	}
}

// SetNoCacheHeaders middleware prevents caching of API responses
func SetNoCacheHeaders(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		c.Response().Header().Set("Cache-Control", "no-store")
		return next(c)
	}
}

// SetJSONContentType middleware ensures all responses have JSON content type
func SetJSONContentType(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		return next(c)
	}
}
