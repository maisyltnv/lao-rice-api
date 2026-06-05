package database

import (
	"fmt"
	"log"
	"time"

	"shopapi/internal/model"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const dbConnectRetries = 30

// New opens a PostgreSQL connection with GORM, runs migrations, and returns the DB handle.
func New(dsn string) (*gorm.DB, error) {
	var db *gorm.DB
	var err error
	for attempt := 1; attempt <= dbConnectRetries; attempt++ {
		db, err = open(dsn)
		if err == nil {
			log.Println("database connected and migrations applied")
			return db, nil
		}
		if attempt < dbConnectRetries {
			log.Printf("database connect attempt %d/%d failed: %v", attempt, dbConnectRetries, err)
			time.Sleep(time.Second)
		}
	}
	return nil, fmt.Errorf("database after %d attempts: %w", dbConnectRetries, err)
}

func open(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get sql db: %w", err)
	}
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)
	sqlDB.SetConnMaxIdleTime(5 * time.Minute)

	if err := db.AutoMigrate(
		&model.User{},
		&model.Category{},
		&model.Product{},
		&model.Order{},
		&model.OrderItem{},
		&model.ExchangeRateConfig{},
		&model.Banner{},
		&model.ShopSettings{},
	); err != nil {
		return nil, fmt.Errorf("auto migrate: %w", err)
	}

	// Backfill order_number for rows created before checkout fields were added.
	if err := db.Exec(`
		UPDATE orders
		SET order_number = 'ORD-' || LPAD(id::text, 8, '0')
		WHERE order_number IS NULL OR order_number = ''
	`).Error; err != nil {
		return nil, fmt.Errorf("backfill order_number: %w", err)
	}
	// Map legacy statuses to the current workflow labels.
	if err := db.Exec(`
		UPDATE orders SET status = 'processing' WHERE status = 'paid';
		UPDATE orders SET status = 'delivered' WHERE status = 'completed';
	`).Error; err != nil {
		return nil, fmt.Errorf("migrate order status: %w", err)
	}

	return db, nil
}
