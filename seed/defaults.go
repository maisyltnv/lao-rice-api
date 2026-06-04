// Package seed holds default data applied by cmd/seed (idempotent).
package seed

// Default admin for VPS / first install (override via env in cmd/seed).
const (
	DefaultAdminUsername = "admin"
	DefaultAdminPassword = "1234"
)
