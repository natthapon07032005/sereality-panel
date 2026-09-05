package middleware

import "github.com/gin-gonic/gin"

// SecurityHeadersMiddleware adds response headers that are safe for both the
// legacy panel UI and the JSON API. Content-Security-Policy is intentionally
// not set here because the legacy templates contain inline scripts; tightening
// CSP needs a separate asset migration.
func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Permissions-Policy", "camera=(), geolocation=(), microphone=()")
		c.Next()
	}
}
