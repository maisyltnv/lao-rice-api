package model

import (
	"time"

	"gorm.io/gorm"
)

// Product is a sellable item with CNY sourcing and LAK retail pricing.
type Product struct {
	ID                uint64    `gorm:"primaryKey" json:"id"`
	Name              string    `gorm:"size:255;not null" json:"name"`
	Description       string    `gorm:"type:text" json:"description"`
	ImageURL          string    `gorm:"size:2048" json:"image_url"`
	ImageURLsJSON     string    `gorm:"type:text" json:"-"`
	ImageURLs         []string  `gorm:"-" json:"image_urls"`
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

func (p *Product) AfterFind(_ *gorm.DB) error {
	p.ImageURLs = ParseProductImageURLs(p.ImageURLsJSON, p.ImageURL)
	if len(p.ImageURLs) > 0 {
		p.ImageURL = p.ImageURLs[0]
	}
	return nil
}

func (p *Product) BeforeSave(_ *gorm.DB) error {
	urls := NormalizeProductImageURLs(p.ImageURLs, p.ImageURL)
	p.ImageURLs = urls
	if len(urls) > 0 {
		p.ImageURL = urls[0]
	} else {
		p.ImageURL = ""
	}
	p.ImageURLsJSON = MarshalProductImageURLs(urls)
	return nil
}
