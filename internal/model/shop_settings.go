package model

import "time"

// ShopSettings is the singleton shop configuration row (id = 1).
type ShopSettings struct {
	ID                         uint64    `gorm:"primaryKey" json:"id"`
	ShippingFeeLAK             float64   `gorm:"type:decimal(18,4);not null" json:"shipping_fee_lak"`
	FreeShippingMinSubtotalLAK float64   `gorm:"type:decimal(18,4);not null" json:"free_shipping_min_subtotal_lak"`
	BcelQrEnabled              bool      `gorm:"not null;default:true" json:"bcel_qr_enabled"`
	CodEnabled                 bool      `gorm:"not null;default:true" json:"cod_enabled"`
	UpdatedAt                  time.Time `json:"updated_at"`
}

func (ShopSettings) TableName() string {
	return "shop_settings"
}
