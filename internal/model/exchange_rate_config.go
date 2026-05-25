package model

import "time"

// ExchangeRateConfig is the shop-wide CNY → LAK rate (single row, id = 1).
type ExchangeRateConfig struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	RateLAK   float64   `gorm:"type:decimal(18,8);not null" json:"rate_lak_per_cny"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (ExchangeRateConfig) TableName() string {
	return "exchange_rate_configs"
}
