package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"shopapi/internal/delivery"
	"shopapi/internal/middleware"
	"shopapi/internal/model"
	"shopapi/internal/service"
	"shopapi/internal/upload"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type OrderHandler struct {
	orders   *service.OrderService
	receipts *upload.PaymentReceiptStore
}

func NewOrderHandler(orders *service.OrderService, receipts *upload.PaymentReceiptStore) *OrderHandler {
	return &OrderHandler{orders: orders, receipts: receipts}
}

type orderLineRequest struct {
	ProductID uint64 `json:"product_id" binding:"required"`
	Quantity  int    `json:"quantity" binding:"required,min=1,max=9999"`
}

type shippingRequest struct {
	RecipientName string  `json:"recipient_name" binding:"required"`
	Phone         string  `json:"phone" binding:"required"`
	Province      string  `json:"province"`
	AddressDetail string  `json:"address_detail" binding:"required"`
	Latitude      float64 `json:"latitude" binding:"required"`
	Longitude     float64 `json:"longitude" binding:"required"`
}

type placeOrderRequest struct {
	Items             []orderLineRequest `json:"items" binding:"required,min=1,dive"`
	Shipping          shippingRequest    `json:"shipping" binding:"required"`
	PaymentMethod     string             `json:"payment_method" binding:"required,oneof=bcel_qr cod"`
	PaymentReceiptURL string             `json:"payment_receipt_url"`
}

// ListByPhone is a public endpoint for customers to track orders by shipping phone (no JWT).
// Newest orders first. Pagination: ?phone=...&page=1&limit=10
func (h *OrderHandler) ListByPhone(c *gin.Context) {
	phone := c.Query("phone")
	if phone == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "phone query parameter is required"})
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	res, err := h.orders.ListByPhone(c.Request.Context(), phone, page, limit)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *OrderHandler) ShippingConfig(c *gin.Context) {
	c.JSON(http.StatusOK, h.orders.ShippingConfig())
}

func (h *OrderHandler) QuoteShipping(c *gin.Context) {
	subtotal, err := strconv.ParseFloat(c.DefaultQuery("subtotal_lak", "0"), 64)
	if err != nil || subtotal < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subtotal_lak"})
		return
	}
	lat, _ := strconv.ParseFloat(c.Query("latitude"), 64)
	lng, _ := strconv.ParseFloat(c.Query("longitude"), 64)
	if lat != 0 || lng != 0 {
		if err := delivery.ValidateCoordinates(lat, lng); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}
	c.JSON(http.StatusOK, h.orders.QuoteShipping(subtotal))
}

// Place creates an order (public). Accepts JSON or multipart/form-data (with payment_receipt file).
func (h *OrderHandler) Place(c *gin.Context) {
	ct := strings.ToLower(c.GetHeader("Content-Type"))
	if strings.HasPrefix(ct, "multipart/form-data") {
		h.placeMultipart(c)
		return
	}
	var req placeOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	h.placeFromRequest(c, req, "")
}

// ListMine returns paginated orders for the authenticated customer (JWT user_id).
func (h *OrderHandler) ListMine(c *gin.Context) {
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
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	res, err := h.orders.ListMinePaginated(c.Request.Context(), uid, page, limit)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}

// List returns all orders (admin only).
func (h *OrderHandler) List(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	items, total, err := h.orders.ListAll(c.Request.Context(), limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": total})
}

// Get returns one order (own order for customers; any order for admin).
type updateOrderStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

// UpdateStatus sets order status (admin): pending | processing | shipped | delivered.
func (h *OrderHandler) UpdateStatus(c *gin.Context) {
	id, err := parseUintParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req updateOrderStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	status, err := service.ParseOrderStatus(req.Status)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	o, err := h.orders.UpdateStatus(c.Request.Context(), id, status)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, o)
}

func (h *OrderHandler) Get(c *gin.Context) {
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
	id, err := parseUintParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	role, _ := c.Get(middleware.ContextRoleKey)
	roleStr, _ := role.(string)

	var o *model.Order
	if roleStr == model.RoleAdmin {
		o, err = h.orders.GetByID(c.Request.Context(), id)
	} else {
		o, err = h.orders.GetMine(c.Request.Context(), uid, id)
	}
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, o)
}

// GetSourceLinks returns procurement URLs per line item (admin only).
func (h *OrderHandler) GetSourceLinks(c *gin.Context) {
	id, err := parseUintParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	res, err := h.orders.SourceLinks(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}
