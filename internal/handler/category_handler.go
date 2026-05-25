package handler

import (
	"errors"
	"net/http"
	"strconv"

	"shopapi/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type CategoryHandler struct {
	svc *service.CategoryService
}

func NewCategoryHandler(svc *service.CategoryService) *CategoryHandler {
	return &CategoryHandler{svc: svc}
}

func (h *CategoryHandler) List(c *gin.Context) {
	rootsOnly := c.Query("roots_only") == "true"
	var parentID *uint64
	if v := c.Query("parent_id"); v != "" {
		id, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid parent_id"})
			return
		}
		parentID = &id
	}
	items, err := h.svc.ListPublic(c.Request.Context(), rootsOnly, parentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *CategoryHandler) Get(c *gin.Context) {
	id, err := parseUintParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	cat, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !cat.IsActive {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, cat)
}

func (h *CategoryHandler) GetBySlug(c *gin.Context) {
	slug := c.Param("slug")
	if slug == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing slug"})
		return
	}
	cat, err := h.svc.GetBySlug(c.Request.Context(), slug)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !cat.IsActive {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, cat)
}

type createCategoryRequest struct {
	ParentID    *uint64 `json:"parent_id"`
	Name        string  `json:"name" binding:"required,min=1,max=128"`
	Slug        string  `json:"slug"`
	Description string  `json:"description"`
	SortOrder   int     `json:"sort_order"`
	IsActive    *bool   `json:"is_active"`
}

func (h *CategoryHandler) Create(c *gin.Context) {
	var req createCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	cat, err := h.svc.Create(c.Request.Context(), service.CreateCategoryInput{
		ParentID:    req.ParentID,
		Name:        req.Name,
		Slug:        req.Slug,
		Description: req.Description,
		SortOrder:   req.SortOrder,
		IsActive:    req.IsActive,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, cat)
}

type updateCategoryRequest struct {
	ClearParent bool    `json:"clear_parent"`
	ParentID    *uint64 `json:"parent_id"`
	Name        *string `json:"name"`
	Slug        *string `json:"slug"`
	Description *string `json:"description"`
	SortOrder   *int    `json:"sort_order"`
	IsActive    *bool   `json:"is_active"`
}

func (h *CategoryHandler) Update(c *gin.Context) {
	id, err := parseUintParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req updateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	cat, err := h.svc.Update(c.Request.Context(), id, service.UpdateCategoryInput{
		ClearParent: req.ClearParent,
		ParentID:    req.ParentID,
		Name:        req.Name,
		Slug:        req.Slug,
		Description: req.Description,
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
	c.JSON(http.StatusOK, cat)
}

func (h *CategoryHandler) Delete(c *gin.Context) {
	id, err := parseUintParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}
