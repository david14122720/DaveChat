package middleware

import (
	"github.com/labstack/echo/v4"
)

// SecurityHeadersMiddleware returns an Echo middleware that sets security
// response headers on every HTTP response. When devMode is true, the middleware
// is a no-op (allows unrestricted development).
//
// Headers set in production:
//   - Content-Security-Policy-Report-Only
//   - Strict-Transport-Security: max-age=31536000; includeSubDomains
//   - X-Frame-Options: DENY
//   - X-Content-Type-Options: nosniff
//   - Referrer-Policy: strict-origin-when-cross-origin
//   - Permissions-Policy: camera=(), microphone=(), geolocation=()
func SecurityHeadersMiddleware(devMode bool) echo.MiddlewareFunc {
	if devMode {
		return func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c echo.Context) error {
				return next(c)
			}
		}
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Response().Header().Set("Content-Security-Policy-Report-Only",
				"default-src 'self'; img-src 'self' data: blob:; media-src 'self'; connect-src 'self' ws: wss:; style-src 'self' 'unsafe-inline'; script-src 'self'")
			c.Response().Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			c.Response().Header().Set("X-Frame-Options", "DENY")
			c.Response().Header().Set("X-Content-Type-Options", "nosniff")
			c.Response().Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			c.Response().Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
			return next(c)
		}
	}
}
