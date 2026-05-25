package middleware

import (
	"net/http"
	"strings"

	"shopapi/internal/service"

	"github.com/gin-gonic/gin"
)

const (
	ContextUserIDKey   = "userID"
	ContextUsernameKey = "username"
	ContextRoleKey     = "role"
)

// JWTAuth validates Authorization: Bearer <token> and attaches user claims to the Gin context.
func JWTAuth(auth *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		lower := strings.ToLower(h)
		if h == "" || !strings.HasPrefix(lower, "bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return
		}
		raw := strings.TrimSpace(h[len("bearer "):])
		if raw == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return
		}
		claims, err := auth.ParseToken(raw)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}
		c.Set(ContextUserIDKey, claims.UserID)
		c.Set(ContextUsernameKey, claims.Username)
		c.Set(ContextRoleKey, claims.Role)
		c.Next()
	}
}

// OptionalJWTAuth attaches claims when a valid Bearer token is sent; does not require auth.
func OptionalJWTAuth(auth *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		lower := strings.ToLower(h)
		if h == "" || !strings.HasPrefix(lower, "bearer ") {
			c.Next()
			return
		}
		raw := strings.TrimSpace(h[len("bearer "):])
		if raw == "" {
			c.Next()
			return
		}
		claims, err := auth.ParseToken(raw)
		if err != nil {
			c.Next()
			return
		}
		c.Set(ContextUserIDKey, claims.UserID)
		c.Set(ContextUsernameKey, claims.Username)
		c.Set(ContextRoleKey, claims.Role)
		c.Next()
	}
}
