package seed

import (
	"context"
	"errors"
	"log"
	"os"

	"shopapi/internal/model"
	"shopapi/internal/repository"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// EnsureDefaultAdmin creates or updates the default admin user (role=admin).
func EnsureDefaultAdmin(ctx context.Context, db *gorm.DB) error {
	username := DefaultAdminUsername
	password := DefaultAdminPassword
	if v := os.Getenv("SEED_ADMIN_USERNAME"); v != "" {
		username = v
	}
	if v := os.Getenv("SEED_ADMIN_PASSWORD"); v != "" {
		password = v
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	repo := repository.NewUserRepository(db)
	existing, err := repo.GetByUsername(ctx, username)
	if err == nil {
		existing.PasswordHash = string(hash)
		existing.Role = model.RoleAdmin
		if err := repo.Save(ctx, existing); err != nil {
			return err
		}
		log.Printf("admin user updated: %s", username)
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	u := &model.User{
		Username:     username,
		PasswordHash: string(hash),
		Role:         model.RoleAdmin,
	}
	if err := repo.Create(ctx, u); err != nil {
		return err
	}
	log.Printf("admin user created: %s", username)
	return nil
}
