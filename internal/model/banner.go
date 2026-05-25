package model

import "time"

// Banner is a homepage hero slide (image, copy, CTA link).
type Banner struct {
	ID          uint64    `gorm:"primaryKey" json:"id"`
	Title       string    `gorm:"size:255;not null" json:"title"`
	Subtitle    string    `gorm:"size:255" json:"subtitle"`
	Description string    `gorm:"type:text" json:"description"`
	ImageURL    string    `gorm:"size:2048;not null" json:"image_url"`
	CTALabel    string    `gorm:"size:128" json:"cta_label"`
	LinkURL     string    `gorm:"size:2048" json:"link_url"`
	SortOrder   int       `gorm:"not null;default:0" json:"sort_order"`
	IsActive    bool      `gorm:"not null;default:true" json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (Banner) TableName() string {
	return "banners"
}
