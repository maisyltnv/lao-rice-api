package model

import "time"

// Role values stored on users.role.
const (
	RoleUser  = "user"
	RoleAdmin = "admin"
)

// User represents an application account.
type User struct {
	ID           uint64    `gorm:"primaryKey" json:"id"`
	Username     string    `gorm:"uniqueIndex;size:64;not null" json:"username"`
	PasswordHash string    `gorm:"size:255;not null" json:"-"`
	Role         string    `gorm:"size:32;not null;default:user" json:"role"`
	// Default shipping snapshot for checkout (customer accounts).
	RecipientName     string  `gorm:"size:128" json:"recipient_name"`
	ShippingPhone     string  `gorm:"size:32" json:"shipping_phone"`
	Province          string  `gorm:"size:128" json:"province"`
	AddressDetail     string  `gorm:"type:text" json:"address_detail"`
	DeliveryLatitude  float64 `json:"delivery_latitude"`
	DeliveryLongitude float64 `json:"delivery_longitude"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

func (User) TableName() string {
	return "users"
}
