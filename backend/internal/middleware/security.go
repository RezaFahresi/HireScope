package middleware

import (
	"github.com/gin-gonic/gin"
)

// SecurityHeaders adds standard defensive HTTP security headers to all responses.
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Permissions-Policy", "geolocation=(), camera=(), microphone=()")

		c.Next()
	}
}
