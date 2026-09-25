package middleware

import (
	"net/http"
	"strings"

	"hirescope/backend/internal/handler"
	"hirescope/backend/internal/repository"
	"hirescope/backend/internal/service"

	"github.com/gin-gonic/gin"
)

const (
	ContextUserID    = "userID"
	ContextUserEmail = "userEmail"
	ContextUserRole  = "userRole"
	ContextRawToken  = "rawToken"
)

// AuthMiddleware creates a Gin middleware that validates the real JWT in the Authorization header.
func AuthMiddleware(jwtService service.JWTService, tokenRepo repository.RevokedTokenRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			handler.RespondError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing authorization header")
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			handler.RespondError(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid authorization header format, expected 'Bearer <token>'")
			c.Abort()
			return
		}

		rawToken := strings.TrimSpace(parts[1])
		if rawToken == "" {
			handler.RespondError(c, http.StatusUnauthorized, "UNAUTHORIZED", "token cannot be empty")
			c.Abort()
			return
		}

		// Check if token was revoked via logout
		tokenHash := jwtService.HashToken(rawToken)
		isRevoked, err := tokenRepo.IsRevoked(c.Request.Context(), tokenHash)
		if err != nil {
			handler.RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to verify token status")
			c.Abort()
			return
		}
		if isRevoked {
			handler.RespondError(c, http.StatusUnauthorized, "TOKEN_REVOKED", "token has been revoked")
			c.Abort()
			return
		}

		// Validate real cryptographically signed JWT with standard library
		claims, err := jwtService.ValidateToken(rawToken)
		if err != nil {
			handler.RespondError(c, http.StatusUnauthorized, "UNAUTHORIZED", err.Error())
			c.Abort()
			return
		}

		// Store authenticated user information in Gin context
		c.Set(ContextUserID, claims.Subject)
		c.Set(ContextUserEmail, claims.Email)
		c.Set(ContextUserRole, string(claims.Role))
		c.Set(ContextRawToken, rawToken)

		c.Next()
	}
}
