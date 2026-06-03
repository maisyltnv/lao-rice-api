package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"shopapi/internal/middleware"
	"shopapi/internal/model"
	"shopapi/internal/service"

	"github.com/gin-gonic/gin"
)

func (h *OrderHandler) placeFromRequest(c *gin.Context, req placeOrderRequest, receiptURL string) {
	uidVal, ok := c.Get(middleware.ContextUserIDKey)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "login required"})
		return
	}
	userID, ok := uidVal.(uint64)
	if !ok || userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "login required"})
		return
	}

	lines := make([]service.OrderLineInput, 0, len(req.Items))
	for _, it := range req.Items {
		lines = append(lines, service.OrderLineInput{ProductID: it.ProductID, Quantity: it.Quantity})
	}
	receipt := receiptURL
	if receipt == "" {
		receipt = strings.TrimSpace(req.PaymentReceiptURL)
	}
	o, err := h.orders.Place(c.Request.Context(), service.PlaceOrderInput{
		UserID: userID,
		Lines:  lines,
		Shipping: service.ShippingInput{
			RecipientName: req.Shipping.RecipientName,
			Phone:         req.Shipping.Phone,
			Province:      req.Shipping.Province,
			AddressDetail: req.Shipping.AddressDetail,
			Latitude:      req.Shipping.Latitude,
			Longitude:     req.Shipping.Longitude,
		},
		PaymentMethod:     req.PaymentMethod,
		PaymentReceiptURL: receipt,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, o)
}

func (h *OrderHandler) placeMultipart(c *gin.Context) {
	if h.receipts == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "upload store not configured"})
		return
	}

	itemsJSON := c.PostForm("items")
	shippingJSON := c.PostForm("shipping")
	paymentMethod := strings.TrimSpace(c.PostForm("payment_method"))
	if itemsJSON == "" || shippingJSON == "" || paymentMethod == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "items, shipping, and payment_method are required"})
		return
	}

	var req placeOrderRequest
	req.PaymentMethod = paymentMethod
	if err := json.Unmarshal([]byte(itemsJSON), &req.Items); err != nil || len(req.Items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid items JSON"})
		return
	}
	if err := json.Unmarshal([]byte(shippingJSON), &req.Shipping); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid shipping JSON"})
		return
	}
	if req.Shipping.RecipientName == "" || req.Shipping.Phone == "" || req.Shipping.AddressDetail == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "shipping fields are required"})
		return
	}
	if req.Shipping.Latitude == 0 && req.Shipping.Longitude == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "delivery location (latitude, longitude) is required"})
		return
	}

	var receiptURL string
	if strings.EqualFold(paymentMethod, model.PaymentMethodBCELQR) {
		file, err := c.FormFile("payment_receipt")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "payment_receipt image is required for bcel_qr"})
			return
		}
		receiptURL, err = h.receipts.Save(file)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	} else if f, err := c.FormFile("payment_receipt"); err == nil && f != nil {
		url, err := h.receipts.Save(f)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		receiptURL = url
	}

	h.placeFromRequest(c, req, receiptURL)
}
