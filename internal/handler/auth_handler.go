package handler

import (
	"net/http"

	"shopapi/internal/middleware"
	"shopapi/internal/service"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	auth *service.AuthService
}

func NewAuthHandler(auth *service.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

type registerRequest struct {
	Username string `json:"username" binding:"required,min=3,max=64"`
	Password string `json:"password" binding:"required,min=8"`
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	u, err := h.auth.Register(c.Request.Context(), service.RegisterInput{
		Username: req.Username,
		Password: req.Password,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"id":       u.ID,
		"username": u.Username,
		"role":     u.Role,
	})
}

type adminRegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=64"`
	Password string `json:"password" binding:"required,min=8"`
}

// AdminRegister creates an admin account for the web admin app (body: username + password only).
func (h *AuthHandler) AdminRegister(c *gin.Context) {
	var req adminRegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	u, err := h.auth.RegisterAdmin(c.Request.Context(), service.RegisterAdminInput{
		Username: req.Username,
		Password: req.Password,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"id":       u.ID,
		"username": u.Username,
		"role":     u.Role,
	})
}

type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	u, tok, err := h.auth.Login(c.Request.Context(), service.LoginInput{
		Username: req.Username,
		Password: req.Password,
	})
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"access_token": tok.Token,
		"expires_at":   tok.ExpiresAt,
		"user": gin.H{
			"id":       u.ID,
			"username": u.Username,
			"role":     u.Role,
		},
	})
}

// AdminLogin is for the web admin panel: only users with role admin receive a token.
func (h *AuthHandler) AdminLogin(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	u, tok, err := h.auth.LoginAdmin(c.Request.Context(), service.LoginInput{
		Username: req.Username,
		Password: req.Password,
	})
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"access_token": tok.Token,
		"expires_at":   tok.ExpiresAt,
		"user": gin.H{
			"id":       u.ID,
			"username": u.Username,
			"role":     u.Role,
		},
	})
}

func (h *AuthHandler) Me(c *gin.Context) {
	uid, _ := c.Get(middleware.ContextUserIDKey)
	name, _ := c.Get(middleware.ContextUsernameKey)
	role, _ := c.Get(middleware.ContextRoleKey)
	c.JSON(http.StatusOK, gin.H{
		"id":       uid,
		"username": name,
		"role":     role,
	})
}

// AdminMe returns the current user only if the JWT is for an admin (for admin SPA).
func (h *AuthHandler) AdminMe(c *gin.Context) {
	h.Me(c)
}
