package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"shopapi/internal/model"
	"shopapi/internal/repository"

	"gorm.io/gorm"
)

var slugNonAlnum = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = slugNonAlnum.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if s == "" {
		return "category"
	}
	return s
}

type CategoryService struct {
	cats     *repository.CategoryRepository
	products *repository.ProductRepository
}

func NewCategoryService(cats *repository.CategoryRepository, products *repository.ProductRepository) *CategoryService {
	return &CategoryService{cats: cats, products: products}
}

type CreateCategoryInput struct {
	ParentID    *uint64
	Name        string
	Slug        string
	Description string
	SortOrder   int
	IsActive    *bool
}

func (s *CategoryService) Create(ctx context.Context, in CreateCategoryInput) (*model.Category, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, errors.New("name is required")
	}
	active := true
	if in.IsActive != nil {
		active = *in.IsActive
	}
	if in.ParentID != nil {
		if _, err := s.cats.GetByID(ctx, *in.ParentID); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, errors.New("parent category not found")
			}
			return nil, err
		}
	}
	slug := strings.TrimSpace(in.Slug)
	if slug == "" {
		slug = slugify(name)
	} else {
		slug = slugify(slug)
	}
	alloc, err := s.allocateSlug(ctx, slug, 0)
	if err != nil {
		return nil, err
	}
	c := &model.Category{
		ParentID:    in.ParentID,
		Name:        name,
		Slug:        alloc,
		Description: strings.TrimSpace(in.Description),
		SortOrder:   in.SortOrder,
		IsActive:    active,
	}
	if err := s.cats.Create(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

type UpdateCategoryInput struct {
	ClearParent   bool
	ParentID      *uint64
	Name          *string
	Slug          *string
	Description   *string
	SortOrder     *int
	IsActive      *bool
}

func (s *CategoryService) Update(ctx context.Context, id uint64, in UpdateCategoryInput) (*model.Category, error) {
	c, err := s.cats.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if in.ClearParent {
		c.ParentID = nil
	} else if in.ParentID != nil {
		pid := *in.ParentID
		if pid == id {
			return nil, errors.New("category cannot be its own parent")
		}
		parent, err := s.cats.GetByID(ctx, pid)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, errors.New("parent category not found")
			}
			return nil, err
		}
		_ = parent
		if s.isDescendant(ctx, id, pid) {
			return nil, errors.New("invalid parent: would create a cycle")
		}
		c.ParentID = in.ParentID
	}
	if in.Name != nil {
		c.Name = strings.TrimSpace(*in.Name)
		if c.Name == "" {
			return nil, errors.New("name cannot be empty")
		}
	}
	if in.Slug != nil {
		slug := slugify(strings.TrimSpace(*in.Slug))
		if slug == "" {
			slug = slugify(c.Name)
		}
		alloc, err := s.allocateSlug(ctx, slug, id)
		if err != nil {
			return nil, err
		}
		c.Slug = alloc
	}
	if in.Description != nil {
		c.Description = strings.TrimSpace(*in.Description)
	}
	if in.SortOrder != nil {
		c.SortOrder = *in.SortOrder
	}
	if in.IsActive != nil {
		c.IsActive = *in.IsActive
	}
	if err := s.cats.Update(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *CategoryService) Delete(ctx context.Context, id uint64) error {
	if _, err := s.cats.GetByID(ctx, id); err != nil {
		return err
	}
	nProd, err := s.products.CountByCategoryID(ctx, id)
	if err != nil {
		return err
	}
	if nProd > 0 {
		return fmt.Errorf("cannot delete category: %d product(s) still assigned", nProd)
	}
	nChild, err := s.cats.CountChildren(ctx, id)
	if err != nil {
		return err
	}
	if nChild > 0 {
		return errors.New("cannot delete category: move or delete child categories first")
	}
	return s.cats.Delete(ctx, id)
}

func (s *CategoryService) GetByID(ctx context.Context, id uint64) (*model.Category, error) {
	return s.cats.GetByID(ctx, id)
}

func (s *CategoryService) GetBySlug(ctx context.Context, slug string) (*model.Category, error) {
	return s.cats.GetBySlug(ctx, slug)
}

// ListPublic lists categories for storefront (active only unless includeInactive).
func (s *CategoryService) ListPublic(ctx context.Context, rootsOnly bool, parentID *uint64) ([]model.Category, error) {
	activeOnly := true
	if rootsOnly {
		return s.cats.ListRoots(ctx, activeOnly)
	}
	return s.cats.List(ctx, activeOnly, parentID)
}

// ListAdmin lists categories including inactive (e.g. back-office).
func (s *CategoryService) ListAdmin(ctx context.Context, parentID *uint64) ([]model.Category, error) {
	return s.cats.List(ctx, false, parentID)
}

func (s *CategoryService) allocateSlug(ctx context.Context, base string, excludeID uint64) (string, error) {
	cand := base
	for n := 0; n < 1000; n++ {
		slug := cand
		if n > 0 {
			slug = fmt.Sprintf("%s-%d", cand, n)
		}
		exists, err := s.cats.SlugExists(ctx, slug, excludeID)
		if err != nil {
			return "", err
		}
		if !exists {
			return slug, nil
		}
	}
	return "", errors.New("could not allocate unique slug")
}

// isDescendant reports whether targetID appears in the ancestor chain starting from startParentID.
func (s *CategoryService) isDescendant(ctx context.Context, targetID, startParentID uint64) bool {
	cur := startParentID
	for cur != 0 {
		if cur == targetID {
			return true
		}
		p, err := s.cats.GetByID(ctx, cur)
		if err != nil || p == nil || p.ParentID == nil {
			break
		}
		cur = *p.ParentID
	}
	return false
}
