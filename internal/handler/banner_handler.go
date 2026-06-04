package handler

import (
	"errors"
	"net/http"
	"strings"

	"shopapi/internal/middleware"
	"shopapi/internal/model"
	"shopapi/internal/service"
	"shopapi/internal/upload"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type BannerHandler struct {
	svc    *service.BannerService
	images *upload.BannerImageStore
}

func NewBannerHandler(svc *service.BannerService, images *upload.BannerImageStore) *BannerHandler {
	return &BannerHandler{svc: svc, images: images}
}

func (h *BannerHandler) maybeDeleteManagedImage(oldURL string) {
	if h.images == nil || oldURL == "" {
		return
	}
	if !h.images.IsManagedURL(oldURL) {
		return
	}
	_ = h.images.DeleteByURL(oldURL)
}

// UploadImage accepts multipart field "image" and returns a public path for image_url.
func (h *BannerHandler) UploadImage(c *gin.Context) {
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

// List returns active banners for the storefront, or all banners for admin (?include_inactive=true).
func (h *BannerHandler) List(c *gin.Context) {
	includeInactive := c.Query("include_inactive") == "true"
	if includeInactive {
		role, _ := c.Get(middleware.ContextRoleKey)
		roleStr, _ := role.(string)
		if roleStr != model.RoleAdmin {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		items, err := h.svc.ListAll(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"items": items})
		return
	}
	items, err := h.svc.ListPublic(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *BannerHandler) Get(c *gin.Context) {
	id, err := parseUintParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	role, _ := c.Get(middleware.ContextRoleKey)
	roleStr, _ := role.(string)

	var b *model.Banner
	if roleStr == model.RoleAdmin {
		b, err = h.svc.GetByID(c.Request.Context(), id)
	} else {
		b, err = h.svc.GetPublic(c.Request.Context(), id)
	}
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, b)
}

type createBannerRequest struct {
	Title       string `json:"title" binding:"required,min=1,max=255"`
	Subtitle    string `json:"subtitle"`
	Description string `json:"description"`
	ImageURL    string `json:"image_url" binding:"required"`
	CTALabel    string `json:"cta_label"`
	LinkURL     string `json:"link_url"`
	SortOrder   int    `json:"sort_order"`
	IsActive    *bool  `json:"is_active"`
}

func (h *BannerHandler) Create(c *gin.Context) {
	var req createBannerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	b, err := h.svc.Create(c.Request.Context(), service.CreateBannerInput{
		Title:       req.Title,
		Subtitle:    req.Subtitle,
		Description: req.Description,
		ImageURL:    req.ImageURL,
		CTALabel:    req.CTALabel,
		LinkURL:     req.LinkURL,
		SortOrder:   req.SortOrder,
		IsActive:    req.IsActive,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, b)
}

type updateBannerRequest struct {
	Title       *string `json:"title"`
	Subtitle    *string `json:"subtitle"`
	Description *string `json:"description"`
	ImageURL    *string `json:"image_url"`
	CTALabel    *string `json:"cta_label"`
	LinkURL     *string `json:"link_url"`
	SortOrder   *int    `json:"sort_order"`
	IsActive    *bool   `json:"is_active"`
}

func (h *BannerHandler) Update(c *gin.Context) {
	id, err := parseUintParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req updateBannerRequest
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
	b, err := h.svc.Update(c.Request.Context(), id, service.UpdateBannerInput{
		Title:       req.Title,
		Subtitle:    req.Subtitle,
		Description: req.Description,
		ImageURL:    req.ImageURL,
		CTALabel:    req.CTALabel,
		LinkURL:     req.LinkURL,
		SortOrder:   req.SortOrder,
		IsActive:    req.IsActive,
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
	c.JSON(http.StatusOK, b)
}

func (h *BannerHandler) Delete(c *gin.Context) {
	id, err := parseUintParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	existing, getErr := h.svc.GetByID(c.Request.Context(), id)
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if getErr == nil {
		h.maybeDeleteManagedImage(existing.ImageURL)
	}
	c.Status(http.StatusNoContent)
}
