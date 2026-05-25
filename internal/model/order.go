package model

import "time"

// OrderStatus is persisted as a string on the order row.
type OrderStatus string

const (
	OrderStatusPending    OrderStatus = "pending"
	OrderStatusProcessing OrderStatus = "processing"
	OrderStatusShipped    OrderStatus = "shipped"
	OrderStatusDelivered  OrderStatus = "delivered"
)

// ValidOrderStatuses lists allowed status values for API validation.
var ValidOrderStatuses = []OrderStatus{
	OrderStatusPending,
	OrderStatusProcessing,
	OrderStatusShipped,
	OrderStatusDelivered,
}

func IsValidOrderStatus(s OrderStatus) bool {
	for _, v := range ValidOrderStatuses {
		if v == s {
			return true
		}
	}
	return false
}

// PaymentMethod values for checkout.
const (
	PaymentMethodBCELQR = "bcel_qr"
	PaymentMethodCOD    = "cod"
)

// Order is a customer purchase in LAK (checkout snapshot).
type Order struct {
	ID                uint64      `gorm:"primaryKey" json:"id"`
	OrderNumber       string      `gorm:"size:32;uniqueIndex;not null" json:"order_number"`
	UserID            uint64      `gorm:"index;not null" json:"user_id"`
	SubtotalLAK       float64     `gorm:"type:decimal(18,4);not null" json:"subtotal_lak"`
	ShippingFeeLAK    float64     `gorm:"type:decimal(18,4);not null" json:"shipping_fee_lak"`
	TotalAmountLAK    float64     `gorm:"type:decimal(18,4);not null" json:"total_amount_lak"`
	Status            OrderStatus `gorm:"type:varchar(32);not null;default:pending" json:"status"`
	PaymentMethod     string      `gorm:"size:32;not null" json:"payment_method"`
	PaymentReceiptURL string      `gorm:"size:2048" json:"payment_receipt_url,omitempty"`
	RecipientName     string      `gorm:"size:128;not null" json:"recipient_name"`
	Phone             string      `gorm:"size:32;not null" json:"phone"`
	Province          string      `gorm:"size:128;not null" json:"province"`
	AddressDetail     string      `gorm:"type:text;not null" json:"address_detail"`
	Latitude          float64     `gorm:"type:decimal(10,7);not null" json:"latitude"`
	Longitude         float64     `gorm:"type:decimal(10,7);not null" json:"longitude"`
	CreatedAt         time.Time   `json:"created_at"`
	UpdatedAt         time.Time   `json:"updated_at"`

	Items []OrderItem `gorm:"foreignKey:OrderID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"items,omitempty"`
}

func (Order) TableName() string {
	return "orders"
}
