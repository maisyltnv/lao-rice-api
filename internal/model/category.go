package model

import "time"

// Category is a catalog grouping for products (hierarchical, URL slug, merchandising flags).
type Category struct {
	ID          uint64    `gorm:"primaryKey" json:"id"`
	ParentID    *uint64   `gorm:"index" json:"parent_id,omitempty"`
	Name        string    `gorm:"size:128;not null" json:"name"`
	Slug        string    `gorm:"size:160;uniqueIndex;not null" json:"slug"`
	Description string    `gorm:"type:text" json:"description,omitempty"`
	SortOrder   int       `gorm:"not null;default:0" json:"sort_order"`
	IsActive    bool      `gorm:"not null;default:true" json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (Category) TableName() string {
	return "categories"
}
