package repository

import (
	"context"

	"shopapi/internal/model"

	"gorm.io/gorm"
)

type CategoryRepository struct {
	db *gorm.DB
}

func NewCategoryRepository(db *gorm.DB) *CategoryRepository {
	return &CategoryRepository{db: db}
}

func (r *CategoryRepository) Create(ctx context.Context, c *model.Category) error {
	return r.db.WithContext(ctx).Create(c).Error
}

func (r *CategoryRepository) GetByID(ctx context.Context, id uint64) (*model.Category, error) {
	var c model.Category
	if err := r.db.WithContext(ctx).First(&c, id).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *CategoryRepository) GetBySlug(ctx context.Context, slug string) (*model.Category, error) {
	var c model.Category
	if err := r.db.WithContext(ctx).Where("slug = ?", slug).First(&c).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

// List returns categories ordered for navigation (sort_order, then name).
func (r *CategoryRepository) List(ctx context.Context, activeOnly bool, parentID *uint64) ([]model.Category, error) {
	var items []model.Category
	q := r.db.WithContext(ctx).Model(&model.Category{})
	if activeOnly {
		q = q.Where("is_active = ?", true)
	}
	if parentID != nil {
		q = q.Where("parent_id = ?", *parentID)
	}
	if err := q.Order("sort_order ASC, name ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// ListRoots returns categories with no parent (top-level aisles).
func (r *CategoryRepository) ListRoots(ctx context.Context, activeOnly bool) ([]model.Category, error) {
	var items []model.Category
	q := r.db.WithContext(ctx).Model(&model.Category{}).Where("parent_id IS NULL")
	if activeOnly {
		q = q.Where("is_active = ?", true)
	}
	if err := q.Order("sort_order ASC, name ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (r *CategoryRepository) SlugExists(ctx context.Context, slug string, excludeID uint64) (bool, error) {
	var n int64
	q := r.db.WithContext(ctx).Model(&model.Category{}).Where("slug = ?", slug)
	if excludeID > 0 {
		q = q.Where("id <> ?", excludeID)
	}
	if err := q.Count(&n).Error; err != nil {
		return false, err
	}
	return n > 0, nil
}

func (r *CategoryRepository) Update(ctx context.Context, c *model.Category) error {
	return r.db.WithContext(ctx).Save(c).Error
}

func (r *CategoryRepository) Delete(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Delete(&model.Category{}, id).Error
}

func (r *CategoryRepository) CountChildren(ctx context.Context, parentID uint64) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&model.Category{}).Where("parent_id = ?", parentID).Count(&n).Error
	return n, err
}
