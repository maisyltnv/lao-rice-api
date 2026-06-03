package handler

import "shopapi/internal/model"

func userJSON(u *model.User) map[string]any {
	phone := u.Username
	if u.ShippingPhone != "" {
		phone = u.ShippingPhone
	}
	return map[string]any{
		"id":                 u.ID,
		"username":           u.Username,
		"phone":              phone,
		"role":               u.Role,
		"recipient_name":     u.RecipientName,
		"shipping_phone":     u.ShippingPhone,
		"province":           u.Province,
		"address_detail":     u.AddressDetail,
		"delivery_latitude":  u.DeliveryLatitude,
		"delivery_longitude": u.DeliveryLongitude,
	}
}
