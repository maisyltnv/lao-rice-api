package handler

import (
	"net/http"

	"shopapi/internal/service"

	"github.com/gin-gonic/gin"
)

type ShopSettingsHandler struct {
	svc *service.ShopSettingsService
}

func NewShopSettingsHandler(svc *service.ShopSettingsService) *ShopSettingsHandler {
	return &ShopSettingsHandler{svc: svc}
}

type updateShopSettingsRequest struct {
	ShippingFeeLAK             float64 `json:"shipping_fee_lak" binding:"gte=0"`
	FreeShippingMinSubtotalLAK float64 `json:"free_shipping_min_subtotal_lak" binding:"gte=0"`
	BcelQrEnabled              bool    `json:"bcel_qr_enabled"`
	CodEnabled                 bool    `json:"cod_enabled"`
}

func (h *ShopSettingsHandler) Get(c *gin.Context) {
	view, err := h.svc.Get(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, view)
}

func (h *ShopSettingsHandler) Update(c *gin.Context) {
	var req updateShopSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	view, err := h.svc.Update(c.Request.Context(), service.UpdateShopSettingsInput{
		ShippingFeeLAK:             req.ShippingFeeLAK,
		FreeShippingMinSubtotalLAK: req.FreeShippingMinSubtotalLAK,
		BcelQrEnabled:              req.BcelQrEnabled,
		CodEnabled:                 req.CodEnabled,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, view)
}
