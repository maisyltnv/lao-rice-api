package model

import "time"

// ShopSettings is the singleton shop configuration row (id = 1).
type ShopSettings struct {
	ID                         uint64    `gorm:"primaryKey" json:"id"`
	ShippingFeeLAK             float64   `gorm:"type:decimal(18,4);not null" json:"shipping_fee_lak"`
	FreeShippingMinSubtotalLAK float64   `gorm:"type:decimal(18,4);not null" json:"free_shipping_min_subtotal_lak"`
	BcelQrEnabled              bool      `gorm:"not null;default:true" json:"bcel_qr_enabled"`
	CodEnabled                 bool      `gorm:"not null;default:true" json:"cod_enabled"`
	ShopName                   string    `gorm:"size:255;not null;default:''" json:"shop_name"`
	Phone                      string    `gorm:"size:64;not null;default:''" json:"phone"`
	Email                      string    `gorm:"size:255;not null;default:''" json:"email"`
	Province                   string    `gorm:"size:128;not null;default:''" json:"province"`
	Address                    string    `gorm:"size:512;not null;default:''" json:"address"`
	Description                string    `gorm:"type:text" json:"description"`
	BankName                   string    `gorm:"size:128;not null;default:''" json:"bank_name"`
	AccountName                string    `gorm:"size:255;not null;default:''" json:"account_name"`
	AccountNumber              string    `gorm:"size:64;not null;default:''" json:"account_number"`
	AdminPrefsJSON             string    `gorm:"type:text" json:"-"`
	UpdatedAt                  time.Time `json:"updated_at"`
}

func (ShopSettings) TableName() string {
	return "shop_settings"
}
