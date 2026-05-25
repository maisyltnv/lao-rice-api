package repository

import (
	"context"

	"shopapi/internal/model"

	"gorm.io/gorm"
)

type BannerRepository struct {
	db *gorm.DB
}

func NewBannerRepository(db *gorm.DB) *BannerRepository {
	return &BannerRepository{db: db}
}

func (r *BannerRepository) Create(ctx context.Context, b *model.Banner) error {
	return r.db.WithContext(ctx).Create(b).Error
}

func (r *BannerRepository) GetByID(ctx context.Context, id uint64) (*model.Banner, error) {
	var b model.Banner
	if err := r.db.WithContext(ctx).First(&b, id).Error; err != nil {
		return nil, err
	}
	return &b, nil
}

func (r *BannerRepository) List(ctx context.Context, activeOnly bool) ([]model.Banner, error) {
	var items []model.Banner
	q := r.db.WithContext(ctx).Model(&model.Banner{})
	if activeOnly {
		q = q.Where("is_active = ?", true)
	}
	err := q.Order("sort_order ASC, id ASC").Find(&items).Error
	return items, err
}

func (r *BannerRepository) Update(ctx context.Context, b *model.Banner) error {
	return r.db.WithContext(ctx).Save(b).Error
}

func (r *BannerRepository) Delete(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Delete(&model.Banner{}, id).Error
}
