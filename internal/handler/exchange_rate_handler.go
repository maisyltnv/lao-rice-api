package handler

import (
	"net/http"

	"shopapi/internal/service"

	"github.com/gin-gonic/gin"
)

type ExchangeRateHandler struct {
	svc *service.ExchangeRateService
}

func NewExchangeRateHandler(svc *service.ExchangeRateService) *ExchangeRateHandler {
	return &ExchangeRateHandler{svc: svc}
}

type setExchangeRateRequest struct {
	RateLAKPerCNY float64 `json:"rate_lak_per_cny" binding:"required,gt=0"`
}

func (h *ExchangeRateHandler) Get(c *gin.Context) {
	view, err := h.svc.Get(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, view)
}

func (h *ExchangeRateHandler) Set(c *gin.Context) {
	var req setExchangeRateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	res, err := h.svc.Set(c.Request.Context(), req.RateLAKPerCNY)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}
