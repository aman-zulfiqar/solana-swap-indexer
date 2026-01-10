package server

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// NotFoundJSON returns a custom HTTP error handler that returns JSON responses
// This ensures all errors (including 404s) have consistent JSON format
func NotFoundJSON() echo.HTTPErrorHandler {
	return func(err error, c echo.Context) {
		// Don't send response if already committed
		if c.Response().Committed {
			return
		}

		// Handle Echo HTTP errors (like 404, 400, etc.)
		if he, ok := err.(*echo.HTTPError); ok {
			_ = c.JSON(he.Code, ErrorResponse{
				Error: http.StatusText(he.Code),
				Code:  he.Code,
			})
			return
		}

		// Handle all other errors as internal server error
		_ = c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "internal server error",
			Code:  http.StatusInternalServerError,
		})
	}
}
