package service

import (
	"context"
	"encoding/json"
	"errors"

	"shopapi/internal/model"
	"shopapi/internal/repository"
)

type AdminPrefsView struct {
	NewOrders     bool `json:"new_orders"`
	LowStock      bool `json:"low_stock"`
	DailySummary  bool `json:"daily_summary"`
	TwoFactor     bool `json:"two_factor"`
	StaffApproval bool `json:"staff_approval"`
}

type ShopSettingsView struct {
	ShippingFeeLAK             float64        `json:"shipping_fee_lak"`
	FreeShippingMinSubtotalLAK float64        `json:"free_shipping_min_subtotal_lak"`
	BcelQrEnabled              bool           `json:"bcel_qr_enabled"`
	CodEnabled                 bool           `json:"cod_enabled"`
	ShopName                   string         `json:"shop_name"`
	Phone                      string         `json:"phone"`
	Email                      string         `json:"email"`
	Province                   string         `json:"province"`
	Address                    string         `json:"address"`
	Description                string         `json:"description"`
	BankName                   string         `json:"bank_name"`
	AccountName                string         `json:"account_name"`
	AccountNumber              string         `json:"account_number"`
	AdminPrefs                 AdminPrefsView `json:"admin_prefs"`
	UpdatedAt                  string         `json:"updated_at,omitempty"`
}

type UpdateShopSettingsInput struct {
	ShippingFeeLAK             float64
	FreeShippingMinSubtotalLAK float64
	BcelQrEnabled              bool
	CodEnabled                 bool
	ShopName                   string
	Phone                      string
	Email                      string
	Province                   string
	Address                    string
	Description                string
	BankName                   string
	AccountName                string
	AccountNumber              string
	AdminPrefs                 AdminPrefsView
}

type ShopSettingsService struct {
	repo       *repository.ShopSettingsRepository
	defaultFee float64
	defaultMin float64
}

func NewShopSettingsService(
	repo *repository.ShopSettingsRepository,
	defaultFee, defaultMin float64,
) *ShopSettingsService {
	return &ShopSettingsService{
		repo:       repo,
		defaultFee: defaultFee,
		defaultMin: defaultMin,
	}
}

func (s *ShopSettingsService) Get(ctx context.Context) (ShopSettingsView, error) {
	cfg, err := s.repo.GetOrCreate(ctx, s.defaultFee, s.defaultMin)
	if err != nil {
		return ShopSettingsView{}, err
	}
	return shopSettingsViewFromModel(cfg), nil
}

func (s *ShopSettingsService) Update(ctx context.Context, in UpdateShopSettingsInput) (ShopSettingsView, error) {
	if in.ShippingFeeLAK < 0 {
		return ShopSettingsView{}, errors.New("shipping_fee_lak must be >= 0")
	}
	if in.FreeShippingMinSubtotalLAK < 0 {
		return ShopSettingsView{}, errors.New("free_shipping_min_subtotal_lak must be >= 0")
	}
	if !in.BcelQrEnabled && !in.CodEnabled {
		return ShopSettingsView{}, errors.New("at least one payment method must be enabled")
	}

	cfg, err := s.repo.GetOrCreate(ctx, s.defaultFee, s.defaultMin)
	if err != nil {
		return ShopSettingsView{}, err
	}
	cfg.ShippingFeeLAK = in.ShippingFeeLAK
	cfg.FreeShippingMinSubtotalLAK = in.FreeShippingMinSubtotalLAK
	cfg.BcelQrEnabled = in.BcelQrEnabled
	cfg.CodEnabled = in.CodEnabled
	cfg.ShopName = in.ShopName
	cfg.Phone = in.Phone
	cfg.Email = in.Email
	cfg.Province = in.Province
	cfg.Address = in.Address
	cfg.Description = in.Description
	cfg.BankName = in.BankName
	cfg.AccountName = in.AccountName
	cfg.AccountNumber = in.AccountNumber
	prefsJSON, err := marshalAdminPrefs(in.AdminPrefs)
	if err != nil {
		return ShopSettingsView{}, err
	}
	cfg.AdminPrefsJSON = prefsJSON
	if err := s.repo.Save(ctx, cfg); err != nil {
		return ShopSettingsView{}, err
	}
	return shopSettingsViewFromModel(cfg), nil
}

func defaultAdminPrefs() AdminPrefsView {
	return AdminPrefsView{
		NewOrders:     true,
		LowStock:      true,
		DailySummary:  false,
		TwoFactor:     false,
		StaffApproval: true,
	}
}

func marshalAdminPrefs(prefs AdminPrefsView) (string, error) {
	raw, err := json.Marshal(prefs)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func parseAdminPrefs(raw string) AdminPrefsView {
	prefs := defaultAdminPrefs()
	if raw == "" {
		return prefs
	}
	if err := json.Unmarshal([]byte(raw), &prefs); err != nil {
		return defaultAdminPrefs()
	}
	return prefs
}

func shopSettingsViewFromModel(cfg *model.ShopSettings) ShopSettingsView {
	updatedAt := ""
	if !cfg.UpdatedAt.IsZero() {
		updatedAt = cfg.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z")
	}
	return ShopSettingsView{
		ShippingFeeLAK:             cfg.ShippingFeeLAK,
		FreeShippingMinSubtotalLAK: cfg.FreeShippingMinSubtotalLAK,
		BcelQrEnabled:              cfg.BcelQrEnabled,
		CodEnabled:                 cfg.CodEnabled,
		ShopName:                   cfg.ShopName,
		Phone:                      cfg.Phone,
		Email:                      cfg.Email,
		Province:                   cfg.Province,
		Address:                    cfg.Address,
		Description:                cfg.Description,
		BankName:                   cfg.BankName,
		AccountName:                cfg.AccountName,
		AccountNumber:              cfg.AccountNumber,
		AdminPrefs:                 parseAdminPrefs(cfg.AdminPrefsJSON),
		UpdatedAt:                  updatedAt,
	}
}
