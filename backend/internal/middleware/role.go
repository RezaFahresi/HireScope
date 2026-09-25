package middleware

import (
	"net/http"

	"hirescope/backend/internal/handler"
	"hirescope/backend/internal/model"

	"github.com/gin-gonic/gin"
)

// RequireRole ensures that the authenticated user possesses at least one of the specified roles.
func RequireRole(allowedRoles ...model.Role) gin.HandlerFunc {
	allowedMap := make(map[model.Role]bool, len(allowedRoles))
	for _, r := range allowedRoles {
		allowedMap[r] = true
	}

	return func(c *gin.Context) {
		roleVal, exists := c.Get(ContextUserRole)
		if !exists {
			handler.RespondError(c, http.StatusUnauthorized, "UNAUTHORIZED", "unauthenticated request")
			c.Abort()
			return
		}

		userRole := model.Role(roleVal.(string))
		if !allowedMap[userRole] {
			handler.RespondError(c, http.StatusForbidden, "FORBIDDEN", "insufficient permissions to access this resource")
			c.Abort()
			return
		}

		c.Next()
	}
}
