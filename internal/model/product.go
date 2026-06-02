package model

import "time"

// Product is a sellable item with CNY sourcing and LAK retail pricing.
type Product struct {
	ID                uint64    `gorm:"primaryKey" json:"id"`
	Name              string    `gorm:"size:255;not null" json:"name"`
	Description       string    `gorm:"type:text" json:"description"`
	ImageURL          string    `gorm:"size:2048" json:"image_url"`
	CategoryID        *uint64   `gorm:"index" json:"category_id,omitempty"`
	Category          *Category `gorm:"foreignKey:CategoryID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"category,omitempty"`
	OriginalPriceCNY  float64   `gorm:"type:decimal(18,4);not null" json:"original_price_cny"`
	ExchangeRate      float64   `gorm:"type:decimal(18,8);not null" json:"exchange_rate"`
	ProfitMargin      float64   `gorm:"type:decimal(10,6);not null" json:"profit_margin"`
	FinalPriceLAK     float64   `gorm:"type:decimal(18,4);not null" json:"final_price_lak"`
	Stock             int       `gorm:"not null;default:0" json:"stock"`
	WeightKg          float64   `gorm:"type:decimal(10,3);not null;default:25" json:"weight_kg"`
	SourceURL         string    `gorm:"size:2048" json:"source_url"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

func (Product) TableName() string {
	return "products"
}
