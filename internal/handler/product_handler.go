package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"shopapi/internal/service"
	"shopapi/internal/upload"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ProductHandler struct {
	svc    *service.ProductService
	images *upload.ProductImageStore
}

func NewProductHandler(svc *service.ProductService, images *upload.ProductImageStore) *ProductHandler {
	return &ProductHandler{svc: svc, images: images}
}

func (h *ProductHandler) maybeDeleteManagedImage(oldURL string) {
	if h.images == nil || oldURL == "" {
		return
	}
	if !h.images.IsManagedURL(oldURL) {
		return
	}
	_ = h.images.DeleteByURL(oldURL)
}

// UploadImage accepts multipart field "image" and returns a public path for image_url.
func (h *ProductHandler) UploadImage(c *gin.Context) {
	if h.images == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "upload store not configured"})
		return
	}
	file, err := c.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "image file is required"})
		return
	}
	url, err := h.images.Save(file)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"image_url": url})
}

type createProductRequest struct {
	Name             string   `json:"name" binding:"required"`
	Description      string   `json:"description"`
	ImageURL         string   `json:"image_url"`
	CategoryID       *uint64  `json:"category_id" binding:"required"`
	OriginalPriceCNY float64  `json:"original_price_cny" binding:"required,gte=0"`
	ExchangeRate     float64  `json:"exchange_rate" binding:"required,gte=0"`
	ProfitMargin     float64  `json:"profit_margin" binding:"required,gte=-1"`
	FinalPriceLAK    *float64 `json:"final_price_lak"`
	Stock            int      `json:"stock" binding:"gte=0"`
	SourceURL        string   `json:"source_url"`
}

func (h *ProductHandler) Create(c *gin.Context) {
	var req createProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	p, err := h.svc.Create(c.Request.Context(), service.CreateProductInput{
		Name:             req.Name,
		Description:      req.Description,
		ImageURL:         req.ImageURL,
		CategoryID:       req.CategoryID,
		OriginalPriceCNY: req.OriginalPriceCNY,
		ExchangeRate:     req.ExchangeRate,
		ProfitMargin:     req.ProfitMargin,
		FinalPriceLAK:    req.FinalPriceLAK,
		Stock:            req.Stock,
		SourceURL:        req.SourceURL,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, p)
}

func (h *ProductHandler) Get(c *gin.Context) {
	id, err := parseUintParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	p, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *ProductHandler) List(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	var categoryID *uint64
	if v := c.Query("category_id"); v != "" {
		id, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid category_id"})
			return
		}
		categoryID = &id
	}
	search := strings.TrimSpace(c.Query("q"))
	if search == "" {
		search = strings.TrimSpace(c.Query("search"))
	}
	items, total, err := h.svc.List(c.Request.Context(), limit, offset, categoryID, search)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": total})
}

type updateProductRequest struct {
	Name             *string  `json:"name"`
	Description      *string  `json:"description"`
	ImageURL         *string  `json:"image_url"`
	ClearCategory    bool     `json:"clear_category"`
	CategoryID       *uint64  `json:"category_id"`
	OriginalPriceCNY *float64 `json:"original_price_cny"`
	ExchangeRate     *float64 `json:"exchange_rate"`
	ProfitMargin     *float64 `json:"profit_margin"`
	FinalPriceLAK    *float64 `json:"final_price_lak"`
	Stock            *int     `json:"stock"`
	SourceURL        *string  `json:"source_url"`
}

func (h *ProductHandler) Update(c *gin.Context) {
	id, err := parseUintParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req updateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	existing, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	p, err := h.svc.Update(c.Request.Context(), id, service.UpdateProductInput{
		Name:             req.Name,
		Description:      req.Description,
		ImageURL:         req.ImageURL,
		ClearCategory:    req.ClearCategory,
		CategoryID:       req.CategoryID,
		OriginalPriceCNY: req.OriginalPriceCNY,
		ExchangeRate:     req.ExchangeRate,
		ProfitMargin:     req.ProfitMargin,
		FinalPriceLAK:    req.FinalPriceLAK,
		Stock:            req.Stock,
		SourceURL:        req.SourceURL,
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.ImageURL != nil {
		newURL := strings.TrimSpace(*req.ImageURL)
		oldURL := strings.TrimSpace(existing.ImageURL)
		if newURL != "" && newURL != oldURL {
			h.maybeDeleteManagedImage(oldURL)
		}
	}
	c.JSON(http.StatusOK, p)
}

func (h *ProductHandler) Delete(c *gin.Context) {
	id, err := parseUintParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	existing, getErr := h.svc.GetByID(c.Request.Context(), id)
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if getErr == nil {
		h.maybeDeleteManagedImage(existing.ImageURL)
	}
	c.Status(http.StatusNoContent)
}

func parseUintParam(s string) (uint64, error) {
	return strconv.ParseUint(s, 10, 64)
}
