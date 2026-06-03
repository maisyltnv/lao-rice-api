package handler

import (
	"errors"
	"net/http"

	"shopapi/internal/middleware"
	"shopapi/internal/service"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	auth *service.AuthService
	otp  *service.OTPService
}

func NewAuthHandler(auth *service.AuthService, otp *service.OTPService) *AuthHandler {
	return &AuthHandler{auth: auth, otp: otp}
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
		"phone":    name,
		"role":     role,
	})
}

// AdminMe returns the current user only if the JWT is for an admin (for admin SPA).
func (h *AuthHandler) AdminMe(c *gin.Context) {
	h.Me(c)
}

type otpSendRequest struct {
	Phone string `json:"phone" binding:"required"`
}

// SendOTP stores a one-time code for the phone (stub SMS — use OTP_STUB_CODE env, default 1234).
func (h *AuthHandler) SendOTP(c *gin.Context) {
	var req otpSendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.otp.Send(req.Phone); err != nil {
		if errors.Is(err, service.ErrInvalidPhone) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid phone number"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "otp sent",
		"phone":   service.NormalizePhone(req.Phone),
	})
}

type otpVerifyRequest struct {
	Phone string `json:"phone" binding:"required"`
	Code  string `json:"code" binding:"required"`
}

// VerifyOTP validates the code and returns a customer JWT (creates account on first login).
func (h *AuthHandler) VerifyOTP(c *gin.Context) {
	var req otpVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.otp.Verify(req.Phone, req.Code); err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidPhone):
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid phone number"})
		case errors.Is(err, service.ErrOTPNotFound):
			c.JSON(http.StatusBadRequest, gin.H{"error": "otp expired or not sent"})
		case errors.Is(err, service.ErrOTPInvalid):
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid otp code"})
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}
	phone := service.NormalizePhone(req.Phone)
	u, err := h.auth.FindOrCreatePhoneUser(c.Request.Context(), phone)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	tok, err := h.auth.IssueTokenForUser(u)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"access_token": tok.Token,
		"expires_at":   tok.ExpiresAt,
		"user": gin.H{
			"id":       u.ID,
			"username": u.Username,
			"phone":    u.Username,
			"role":     u.Role,
		},
	})
}
