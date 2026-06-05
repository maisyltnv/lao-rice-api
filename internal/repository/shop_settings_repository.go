package repository

import (
	"context"
	"errors"

	"shopapi/internal/model"

	"gorm.io/gorm"
)

type ShopSettingsRepository struct {
	db *gorm.DB
}

func NewShopSettingsRepository(db *gorm.DB) *ShopSettingsRepository {
	return &ShopSettingsRepository{db: db}
}

// GetOrCreate returns the singleton settings row, seeding defaults if missing.
func (r *ShopSettingsRepository) GetOrCreate(
	ctx context.Context,
	defaultFee, defaultFreeMin float64,
) (*model.ShopSettings, error) {
	var cfg model.ShopSettings
	err := r.db.WithContext(ctx).First(&cfg, 1).Error
	if err == nil {
		return &cfg, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	cfg = model.ShopSettings{
		ID:                         1,
		ShippingFeeLAK:             defaultFee,
		FreeShippingMinSubtotalLAK: defaultFreeMin,
		BcelQrEnabled:              true,
		CodEnabled:                 true,
	}
	if err := r.db.WithContext(ctx).Create(&cfg).Error; err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (r *ShopSettingsRepository) Save(ctx context.Context, cfg *model.ShopSettings) error {
	return r.db.WithContext(ctx).Save(cfg).Error
}
