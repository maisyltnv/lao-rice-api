package config

import (
	"os"
	"strconv"
)

// Config holds runtime configuration loaded from the environment.
type Config struct {
	Port                      string
	DatabaseURL               string
	JWTSecret                 string
	JWTExpiryH                int
	ShippingFeeLAK             float64
	FreeShippingMinSubtotalLAK float64
	UploadDir                  string
	UploadURLPrefix            string
	ImagesDir                  string
}

// Load reads configuration from environment variables with sensible defaults for local dev.
func Load() Config {
	expiry := 72
	if v := os.Getenv("JWT_EXPIRY_HOURS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			expiry = n
		}
	}
	return Config{
		Port:                       getenv("PORT", "8080"),
		DatabaseURL:                getenv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/shopapi?sslmode=disable"),
		JWTSecret:                  getenv("JWT_SECRET", "change-me-in-production-use-long-random-secret"),
		JWTExpiryH:                 expiry,
		ShippingFeeLAK:             getenvFloat("SHIPPING_FEE_LAK", 30000),
		FreeShippingMinSubtotalLAK: getenvFloat("FREE_SHIPPING_MIN_SUBTOTAL_LAK", 500000),
		UploadDir:                  getenv("UPLOAD_DIR", "uploads"),
		UploadURLPrefix:            getenv("UPLOAD_URL_PREFIX", "/uploads"),
		ImagesDir:                  getenv("IMAGES_DIR", "images"),
	}
}

func getenvFloat(key string, fallback float64) float64 {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.ParseFloat(v, 64)
	if err != nil || n < 0 {
		return fallback
	}
	return n
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
