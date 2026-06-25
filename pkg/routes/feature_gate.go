package routes

import (
	"github.com/labstack/echo/v5"
)

// RequireFeature returns a no-op middleware. The license system has been removed
// and all features are always available.
func RequireFeature(feature string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			return next(c)
		}
	}
}
