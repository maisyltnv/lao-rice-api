package repository

import (
	"context"
	"errors"

	"shopapi/internal/model"

	"gorm.io/gorm"
)

const defaultExchangeRateLAK = 3000

type ExchangeRateRepository struct {
	db *gorm.DB
}

func NewExchangeRateRepository(db *gorm.DB) *ExchangeRateRepository {
	return &ExchangeRateRepository{db: db}
}

// GetOrCreate returns the singleton config (id = 1), seeding default rate if missing.
func (r *ExchangeRateRepository) GetOrCreate(ctx context.Context) (*model.ExchangeRateConfig, error) {
	var cfg model.ExchangeRateConfig
	err := r.db.WithContext(ctx).First(&cfg, 1).Error
	if err == nil {
		return &cfg, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	cfg = model.ExchangeRateConfig{ID: 1, RateLAK: defaultExchangeRateLAK}
	if err := r.db.WithContext(ctx).Create(&cfg).Error; err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (r *ExchangeRateRepository) Save(ctx context.Context, cfg *model.ExchangeRateConfig) error {
	return r.db.WithContext(ctx).Save(cfg).Error
}

// Transaction runs fn inside a database transaction.
func (r *ExchangeRateRepository) Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return r.db.WithContext(ctx).Transaction(fn)
}
