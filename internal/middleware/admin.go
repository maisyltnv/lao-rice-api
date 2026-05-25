package middleware

import (
	"net/http"

	"shopapi/internal/model"

	"github.com/gin-gonic/gin"
)

// RequireRole returns middleware that allows the request only if JWT claims include the given role.
func RequireRole(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		v, ok := c.Get(ContextRoleKey)
		if !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		got, _ := v.(string)
		if got != role {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		c.Next()
	}
}

// RequireAdmin is shorthand for RequireRole(model.RoleAdmin).
func RequireAdmin() gin.HandlerFunc {
	return RequireRole(model.RoleAdmin)
}
