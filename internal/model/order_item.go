package model

import "time"

// OrderItem is one line on an order (price snapshot at checkout).
type OrderItem struct {
	ID           uint64    `gorm:"primaryKey" json:"id"`
	OrderID      uint64    `gorm:"index;not null" json:"order_id"`
	ProductID    uint64    `gorm:"index;not null" json:"product_id"`
	ProductName  string    `gorm:"size:255;not null" json:"product_name"`
	UnitPriceLAK float64   `gorm:"type:decimal(18,4);not null" json:"unit_price_lak"`
	Quantity     int       `gorm:"not null" json:"quantity"`
	LineTotalLAK float64   `gorm:"type:decimal(18,4);not null" json:"line_total_lak"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`

	Product *Product `gorm:"foreignKey:ProductID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"product,omitempty"`
}

func (OrderItem) TableName() string {
	return "order_items"
}
