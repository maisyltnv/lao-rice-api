package repository

import (
	"context"
	"errors"
	"fmt"

	"shopapi/internal/model"

	"gorm.io/gorm"
)

// OrderRepository persists orders and order line items.
type OrderRepository struct {
	db *gorm.DB
}

func NewOrderRepository(db *gorm.DB) *OrderRepository {
	return &OrderRepository{db: db}
}

// CreateWithItems creates an order and its line items in one transaction.
// OrderNumber is assigned after insert using the generated id (e.g. ORD-00000008).
func (r *OrderRepository) CreateWithItems(ctx context.Context, o *model.Order, items []model.OrderItem) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Decrement stock atomically (prevents overselling under concurrency).
		// If a product has insufficient stock, the transaction is rolled back.
		for _, it := range items {
			if it.Quantity <= 0 {
				continue
			}
			res := tx.Model(&model.Product{}).
				Where("id = ? AND stock >= ?", it.ProductID, it.Quantity).
				UpdateColumn("stock", gorm.Expr("stock - ?", it.Quantity))
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected == 0 {
				return errors.New("insufficient stock")
			}
		}

		if err := tx.Create(o).Error; err != nil {
			return err
		}
		orderNumber := formatOrderNumber(o.ID)
		if err := tx.Model(o).Update("order_number", orderNumber).Error; err != nil {
			return err
		}
		o.OrderNumber = orderNumber
		for i := range items {
			items[i].OrderID = o.ID
		}
		if len(items) > 0 {
			if err := tx.Create(&items).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func formatOrderNumber(id uint64) string {
	return fmt.Sprintf("ORD-%08d", id)
}

// GetByID returns an order by primary key with line items.
func (r *OrderRepository) GetByID(ctx context.Context, orderID uint64) (*model.Order, error) {
	var o model.Order
	err := r.db.WithContext(ctx).
		Preload("Items").
		Preload("Items.Product").
		First(&o, orderID).Error
	if err != nil {
		return nil, err
	}
	return &o, nil
}

// UpdateStatus sets the order status.
func (r *OrderRepository) UpdateStatus(ctx context.Context, orderID uint64, status model.OrderStatus) error {
	res := r.db.WithContext(ctx).Model(&model.Order{}).
		Where("id = ?", orderID).
		Update("status", status)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// GetByUser returns one order if it belongs to the user, with line items and product refs.
func (r *OrderRepository) GetByUser(ctx context.Context, orderID, userID uint64) (*model.Order, error) {
	var o model.Order
	err := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", orderID, userID).
		Preload("Items").
		Preload("Items.Product").
		First(&o).Error
	if err != nil {
		return nil, err
	}
	return &o, nil
}

// ListAll returns every order, newest first, with line items (admin).
func (r *OrderRepository) ListAll(ctx context.Context, limit, offset int) ([]model.Order, int64, error) {
	var items []model.Order
	var total int64
	q := r.db.WithContext(ctx).Model(&model.Order{})
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := r.db.WithContext(ctx).
		Preload("Items").
		Preload("Items.Product").
		Order("created_at DESC, id DESC").
		Limit(limit).
		Offset(offset).
		Find(&items).Error
	return items, total, err
}

// ListByUserID returns orders for a user, newest first, with line items.
func (r *OrderRepository) ListByUserID(ctx context.Context, userID uint64, limit, offset int) ([]model.Order, int64, error) {
	var items []model.Order
	var total int64
	q := r.db.WithContext(ctx).Model(&model.Order{}).Where("user_id = ?", userID)
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Preload("Items").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&items).Error
	return items, total, err
}

// ListByPhone returns orders matching the shipping phone (newest first).
func (r *OrderRepository) ListByPhone(ctx context.Context, phone string, limit, offset int) ([]model.Order, int64, error) {
	var items []model.Order
	var total int64
	q := r.db.WithContext(ctx).Model(&model.Order{}).Where("phone = ?", phone)
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := r.db.WithContext(ctx).
		Where("phone = ?", phone).
		Preload("Items").
		Order("created_at DESC, id DESC").
		Limit(limit).
		Offset(offset).
		Find(&items).Error
	return items, total, err
}
