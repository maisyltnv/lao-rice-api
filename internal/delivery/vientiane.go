package delivery

import "errors"

// Vientiane Capital approximate bounds (delivery zone).
const (
	ProvinceName = "ນະຄອນຫຼວງວຽງຈັນ"
	MinLat       = 17.85
	MaxLat       = 18.25
	MinLng       = 102.45
	MaxLng       = 102.85
)

// ValidateCoordinates ensures the drop-off point is inside Vientiane Capital.
func ValidateCoordinates(lat, lng float64) error {
	if lat < MinLat || lat > MaxLat || lng < MinLng || lng > MaxLng {
		return errors.New("delivery location must be within Vientiane Capital (ນະຄອນຫຼວງວຽງຈັນ)")
	}
	return nil
}
