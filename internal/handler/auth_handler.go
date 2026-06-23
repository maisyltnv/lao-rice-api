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
	uidVal, ok := c.Get(middleware.ContextUserIDKey)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	uid, ok := uidVal.(uint64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	u, err := h.auth.GetUserByID(c.Request.Context(), uid)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	c.JSON(http.StatusOK, userJSON(u))
}

type updateProfileRequest struct {
	RecipientName      string  `json:"recipient_name" binding:"required"`
	ShippingPhone      string  `json:"shipping_phone"`
	Province           string  `json:"province"`
	AddressDetail      string  `json:"address_detail" binding:"required"`
	DeliveryLatitude   float64 `json:"delivery_latitude"`
	DeliveryLongitude  float64 `json:"delivery_longitude"`
}

// UpdateProfile saves the customer's default shipping info (prefills checkout).
func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	uidVal, ok := c.Get(middleware.ContextUserIDKey)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	uid, ok := uidVal.(uint64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var req updateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	u, err := h.auth.UpdateCustomerProfile(c.Request.Context(), uid, service.UpdateProfileInput{
		RecipientName:     req.RecipientName,
		ShippingPhone:     req.ShippingPhone,
		Province:          req.Province,
		AddressDetail:     req.AddressDetail,
		DeliveryLatitude:  req.DeliveryLatitude,
		DeliveryLongitude: req.DeliveryLongitude,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, userJSON(u))
}

// DeleteMe permanently deletes the authenticated user's account and personal data.
func (h *AuthHandler) DeleteMe(c *gin.Context) {
	uidVal, ok := c.Get(middleware.ContextUserIDKey)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	uid, ok := uidVal.(uint64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	if err := h.auth.DeleteAccount(c.Request.Context(), uid); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "delete failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": true})
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
		"user":         userJSON(u),
	})
}
