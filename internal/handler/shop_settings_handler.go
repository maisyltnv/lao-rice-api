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

type adminPrefsRequest struct {
	NewOrders     bool `json:"new_orders"`
	LowStock      bool `json:"low_stock"`
	DailySummary  bool `json:"daily_summary"`
	TwoFactor     bool `json:"two_factor"`
	StaffApproval bool `json:"staff_approval"`
}

type updateShopSettingsRequest struct {
	ShippingFeeLAK             float64           `json:"shipping_fee_lak" binding:"gte=0"`
	FreeShippingMinSubtotalLAK float64           `json:"free_shipping_min_subtotal_lak" binding:"gte=0"`
	BcelQrEnabled              bool              `json:"bcel_qr_enabled"`
	CodEnabled                 bool              `json:"cod_enabled"`
	ShopName                   string            `json:"shop_name"`
	Phone                      string            `json:"phone"`
	Email                      string            `json:"email"`
	Province                   string            `json:"province"`
	Address                    string            `json:"address"`
	Description                string            `json:"description"`
	BankName                   string            `json:"bank_name"`
	AccountName                string            `json:"account_name"`
	AccountNumber              string            `json:"account_number"`
	AdminPrefs                 adminPrefsRequest `json:"admin_prefs"`
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
		ShopName:                   req.ShopName,
		Phone:                      req.Phone,
		Email:                      req.Email,
		Province:                   req.Province,
		Address:                    req.Address,
		Description:                req.Description,
		BankName:                   req.BankName,
		AccountName:                req.AccountName,
		AccountNumber:              req.AccountNumber,
		AdminPrefs: service.AdminPrefsView{
			NewOrders:     req.AdminPrefs.NewOrders,
			LowStock:      req.AdminPrefs.LowStock,
			DailySummary:  req.AdminPrefs.DailySummary,
			TwoFactor:     req.AdminPrefs.TwoFactor,
			StaffApproval: req.AdminPrefs.StaffApproval,
		},
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, view)
}
