package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"

	"shopapi/internal/delivery"
	"shopapi/internal/model"
	"shopapi/internal/repository"

	"gorm.io/gorm"
)

const maxOrderQty = 9999

// OrderService handles checkout: orders are built from product lines with server-side pricing.
type OrderService struct {
	orders       *repository.OrderRepository
	products     *repository.ProductRepository
	shopSettings *repository.ShopSettingsRepository
	defaultFee   float64
	defaultMin   float64
}

func NewOrderService(
	orders *repository.OrderRepository,
	products *repository.ProductRepository,
	shopSettings *repository.ShopSettingsRepository,
	defaultFee, defaultMin float64,
) *OrderService {
	return &OrderService{
		orders:       orders,
		products:     products,
		shopSettings: shopSettings,
		defaultFee:   defaultFee,
		defaultMin:   defaultMin,
	}
}

func (s *OrderService) shippingRules(ctx context.Context) (fee float64, freeMin float64, err error) {
	cfg, err := s.shopSettings.GetOrCreate(ctx, s.defaultFee, s.defaultMin)
	if err != nil {
		return s.defaultFee, s.defaultMin, err
	}
	return cfg.ShippingFeeLAK, cfg.FreeShippingMinSubtotalLAK, nil
}

type OrderLineInput struct {
	ProductID uint64
	Quantity  int
}

type ShippingInput struct {
	RecipientName string
	Phone         string
	Province      string
	AddressDetail string
	Latitude      float64
	Longitude     float64
}

type PlaceOrderInput struct {
	UserID            uint64
	Lines             []OrderLineInput
	Shipping          ShippingInput
	PaymentMethod     string
	PaymentReceiptURL string
}

type ShippingConfigView struct {
	ShippingFeeLAK             float64 `json:"shipping_fee_lak"`
	FreeShippingMinSubtotalLAK float64 `json:"free_shipping_min_subtotal_lak"`
	BcelQrEnabled              bool    `json:"bcel_qr_enabled"`
	CodEnabled                 bool    `json:"cod_enabled"`
}

// ShippingConfig returns shop shipping rules for the checkout UI.
func (s *OrderService) ShippingConfig(ctx context.Context) (ShippingConfigView, error) {
	cfg, err := s.shopSettings.GetOrCreate(ctx, s.defaultFee, s.defaultMin)
	if err != nil {
		return ShippingConfigView{}, err
	}
	return ShippingConfigView{
		ShippingFeeLAK:             cfg.ShippingFeeLAK,
		FreeShippingMinSubtotalLAK: cfg.FreeShippingMinSubtotalLAK,
		BcelQrEnabled:              cfg.BcelQrEnabled,
		CodEnabled:                 cfg.CodEnabled,
	}, nil
}

type ShippingQuoteInput struct {
	SubtotalLAK float64
}

type ShippingQuoteView struct {
	SubtotalLAK               float64 `json:"subtotal_lak"`
	ShippingFeeLAK            float64 `json:"shipping_fee_lak"`
	TotalAmountLAK            float64 `json:"total_amount_lak"`
	FreeShippingMinSubtotalLAK float64 `json:"free_shipping_min_subtotal_lak"`
	AmountUntilFreeShippingLAK float64 `json:"amount_until_free_shipping_lak"`
	FreeShippingApplied       bool    `json:"free_shipping_applied"`
}

// QuoteShipping estimates fees for a cart subtotal (matches checkout summary sidebar).
func (s *OrderService) QuoteShipping(ctx context.Context, subtotalLAK float64) (ShippingQuoteView, error) {
	subtotal := roundMoneyLAK(subtotalLAK)
	fee, freeMin, err := s.shippingRules(ctx)
	if err != nil {
		return ShippingQuoteView{}, err
	}
	shippingFee := s.calcShippingFee(subtotal, fee, freeMin)
	total := roundMoneyLAK(subtotal + shippingFee)
	untilFree := 0.0
	if subtotal < freeMin {
		untilFree = roundMoneyLAK(freeMin - subtotal)
	}
	return ShippingQuoteView{
		SubtotalLAK:                subtotal,
		ShippingFeeLAK:             shippingFee,
		TotalAmountLAK:             total,
		FreeShippingMinSubtotalLAK: freeMin,
		AmountUntilFreeShippingLAK: untilFree,
		FreeShippingApplied:        shippingFee == 0 && subtotal > 0,
	}, nil
}

func (s *OrderService) calcShippingFee(subtotalLAK, feeLAK, freeMin float64) float64 {
	if subtotalLAK >= freeMin {
		return 0
	}
	return roundMoneyLAK(feeLAK)
}

func roundMoneyLAK(x float64) float64 {
	return math.Round(x*100) / 100
}

func (s *OrderService) validatePaymentMethod(ctx context.Context, method string) error {
	m := strings.ToLower(strings.TrimSpace(method))
	cfg, err := s.shopSettings.GetOrCreate(ctx, s.defaultFee, s.defaultMin)
	if err != nil {
		return err
	}
	switch m {
	case model.PaymentMethodBCELQR:
		if !cfg.BcelQrEnabled {
			return errors.New("bcel_qr payment is disabled")
		}
		return nil
	case model.PaymentMethodCOD:
		if !cfg.CodEnabled {
			return errors.New("cod payment is disabled")
		}
		return nil
	default:
		return errors.New("payment_method must be bcel_qr or cod")
	}
}

// Place creates an order from product lines; pricing and shipping are computed server-side.
func (s *OrderService) Place(ctx context.Context, in PlaceOrderInput) (*model.Order, error) {
	if len(in.Lines) == 0 {
		return nil, errors.New("order must have at least one line item")
	}
	if err := s.validatePaymentMethod(ctx, in.PaymentMethod); err != nil {
		return nil, err
	}
	if strings.EqualFold(strings.TrimSpace(in.PaymentMethod), model.PaymentMethodBCELQR) &&
		strings.TrimSpace(in.PaymentReceiptURL) == "" {
		return nil, errors.New("payment_receipt is required for bcel_qr")
	}
	shipping := in.Shipping
	if strings.TrimSpace(shipping.RecipientName) == "" {
		return nil, errors.New("recipient_name is required")
	}
	if strings.TrimSpace(shipping.Phone) == "" {
		return nil, errors.New("phone is required")
	}
	if strings.TrimSpace(shipping.AddressDetail) == "" {
		return nil, errors.New("address_detail is required")
	}
	if err := delivery.ValidateCoordinates(shipping.Latitude, shipping.Longitude); err != nil {
		return nil, err
	}

	merged := make(map[uint64]int)
	for _, line := range in.Lines {
		if line.ProductID == 0 {
			return nil, errors.New("invalid product_id")
		}
		if line.Quantity < 1 || line.Quantity > maxOrderQty {
			return nil, errors.New("invalid quantity")
		}
		merged[line.ProductID] += line.Quantity
		if merged[line.ProductID] > maxOrderQty {
			return nil, errors.New("quantity per product exceeds limit")
		}
	}

	var items []model.OrderItem
	var subtotal float64
	pids := make([]uint64, 0, len(merged))
	for pid := range merged {
		pids = append(pids, pid)
	}
	sort.Slice(pids, func(i, j int) bool { return pids[i] < pids[j] })
	for _, pid := range pids {
		qty := merged[pid]
		p, err := s.products.GetByID(ctx, pid)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, errors.New("product not found")
			}
			return nil, err
		}
		if qty > p.Stock {
			return nil, fmt.Errorf("insufficient stock for %s", p.Name)
		}
		unit := roundMoneyLAK(p.FinalPriceLAK)
		lineTotal := roundMoneyLAK(unit * float64(qty))
		items = append(items, model.OrderItem{
			ProductID:    pid,
			ProductName:  p.Name,
			UnitPriceLAK: unit,
			Quantity:     qty,
			LineTotalLAK: lineTotal,
		})
		subtotal += lineTotal
	}
	subtotal = roundMoneyLAK(subtotal)
	fee, freeMin, err := s.shippingRules(ctx)
	if err != nil {
		return nil, err
	}
	shippingFee := s.calcShippingFee(subtotal, fee, freeMin)
	total := roundMoneyLAK(subtotal + shippingFee)

	o := &model.Order{
		UserID:            in.UserID,
		SubtotalLAK:       subtotal,
		ShippingFeeLAK:    shippingFee,
		TotalAmountLAK:    total,
		Status:            model.OrderStatusPending,
		PaymentMethod:     strings.ToLower(strings.TrimSpace(in.PaymentMethod)),
		PaymentReceiptURL: strings.TrimSpace(in.PaymentReceiptURL),
		RecipientName:     strings.TrimSpace(shipping.RecipientName),
		Phone:             NormalizePhone(shipping.Phone),
		Province:          delivery.ProvinceName,
		AddressDetail:     strings.TrimSpace(shipping.AddressDetail),
		Latitude:          shipping.Latitude,
		Longitude:         shipping.Longitude,
	}
	if err := s.orders.CreateWithItems(ctx, o, items); err != nil {
		return nil, err
	}
	if in.UserID == 0 {
		return s.orders.GetByID(ctx, o.ID)
	}
	return s.orders.GetByUser(ctx, o.ID, in.UserID)
}

// GetMine returns a single order for the user with items and products.
func (s *OrderService) GetMine(ctx context.Context, userID, orderID uint64) (*model.Order, error) {
	return s.orders.GetByUser(ctx, orderID, userID)
}

// GetByID returns any order by id (admin).
func (s *OrderService) GetByID(ctx context.Context, orderID uint64) (*model.Order, error) {
	return s.orders.GetByID(ctx, orderID)
}

// ParseOrderStatus normalizes client input (case-insensitive).
func ParseOrderStatus(raw string) (model.OrderStatus, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case string(model.OrderStatusPending):
		return model.OrderStatusPending, nil
	case string(model.OrderStatusProcessing):
		return model.OrderStatusProcessing, nil
	case string(model.OrderStatusShipped):
		return model.OrderStatusShipped, nil
	case string(model.OrderStatusDelivered):
		return model.OrderStatusDelivered, nil
	default:
		return "", errors.New("status must be one of: pending, processing, shipped, delivered")
	}
}

// UpdateStatus updates order status (admin).
func (s *OrderService) UpdateStatus(ctx context.Context, orderID uint64, status model.OrderStatus) (*model.Order, error) {
	if !model.IsValidOrderStatus(status) {
		return nil, errors.New("invalid order status")
	}
	if err := s.orders.UpdateStatus(ctx, orderID, status); err != nil {
		return nil, err
	}
	return s.orders.GetByID(ctx, orderID)
}

// NormalizePhone trims spaces; customers may type with or without spaces.
func NormalizePhone(phone string) string {
	p := strings.TrimSpace(phone)
	p = strings.ReplaceAll(p, " ", "")
	p = strings.ReplaceAll(p, "-", "")
	return p
}

// OrderPageResult is a paginated list of orders (newest first).
type OrderPageResult struct {
	Items      []model.Order `json:"items"`
	Page       int           `json:"page"`
	Limit      int           `json:"limit"`
	Total      int64         `json:"total"`
	TotalPages int           `json:"total_pages"`
	HasNext    bool          `json:"has_next"`
	HasPrev    bool          `json:"has_prev"`
}

// ListByPhone returns paginated orders for a shipping phone (newest first).
func (s *OrderService) ListByPhone(ctx context.Context, phone string, page, limit int) (*OrderPageResult, error) {
	phone = NormalizePhone(phone)
	if len(phone) < 8 {
		return nil, errors.New("phone must be at least 8 characters")
	}
	if page < 1 {
		page = 1
	}
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}
	offset := (page - 1) * limit
	items, total, err := s.orders.ListByPhone(ctx, phone, limit, offset)
	if err != nil {
		return nil, err
	}
	totalPages := 0
	if total > 0 {
		totalPages = int((total + int64(limit) - 1) / int64(limit))
	}
	return &OrderPageResult{
		Items:      items,
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
		HasNext:    page < totalPages,
		HasPrev:    page > 1 && totalPages > 0,
	}, nil
}

// ListMine returns paginated orders for the given user (includes line items).
func (s *OrderService) ListMine(ctx context.Context, userID uint64, limit, offset int) ([]model.Order, int64, error) {
	limit, offset = normalizeOrderPagination(limit, offset)
	return s.orders.ListByUserID(ctx, userID, limit, offset)
}

// ListMinePaginated returns orders for the authenticated customer (newest first).
func (s *OrderService) ListMinePaginated(ctx context.Context, userID uint64, page, limit int) (*OrderPageResult, error) {
	if page < 1 {
		page = 1
	}
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}
	offset := (page - 1) * limit
	items, total, err := s.ListMine(ctx, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	totalPages := 0
	if total > 0 {
		totalPages = int((total + int64(limit) - 1) / int64(limit))
	}
	return &OrderPageResult{
		Items:      items,
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
		HasNext:    page < totalPages,
		HasPrev:    page > 1 && totalPages > 0,
	}, nil
}

// ListAll returns paginated orders for every customer (admin).
func (s *OrderService) ListAll(ctx context.Context, limit, offset int) ([]model.Order, int64, error) {
	limit, offset = normalizeOrderPagination(limit, offset)
	return s.orders.ListAll(ctx, limit, offset)
}

// OrderSourceLink is a supplier/procurement URL for one order line (admin).
type OrderSourceLink struct {
	OrderItemID uint64 `json:"order_item_id"`
	ProductID   uint64 `json:"product_id"`
	ProductName string `json:"product_name"`
	Quantity    int    `json:"quantity"`
	SourceURL   string `json:"source_url"`
}

// OrderSourceLinksResult groups procurement links for an order.
type OrderSourceLinksResult struct {
	OrderID     uint64            `json:"order_id"`
	OrderNumber string            `json:"order_number"`
	Links       []OrderSourceLink `json:"links"`
}

// SourceLinks returns supplier URLs for each line on an order (admin procurement).
func (s *OrderService) SourceLinks(ctx context.Context, orderID uint64) (*OrderSourceLinksResult, error) {
	o, err := s.orders.GetByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	links := make([]OrderSourceLink, 0, len(o.Items))
	for _, it := range o.Items {
		url := ""
		if it.Product != nil {
			url = strings.TrimSpace(it.Product.SourceURL)
		}
		links = append(links, OrderSourceLink{
			OrderItemID: it.ID,
			ProductID:   it.ProductID,
			ProductName: it.ProductName,
			Quantity:    it.Quantity,
			SourceURL:   url,
		})
	}
	return &OrderSourceLinksResult{
		OrderID:     o.ID,
		OrderNumber: o.OrderNumber,
		Links:       links,
	}, nil
}

func normalizeOrderPagination(limit, offset int) (int, int) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}
