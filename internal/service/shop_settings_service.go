package service

import (
	"context"
	"errors"

	"shopapi/internal/model"
	"shopapi/internal/repository"
)

type ShopSettingsView struct {
	ShippingFeeLAK             float64 `json:"shipping_fee_lak"`
	FreeShippingMinSubtotalLAK float64 `json:"free_shipping_min_subtotal_lak"`
	BcelQrEnabled              bool    `json:"bcel_qr_enabled"`
	CodEnabled                 bool    `json:"cod_enabled"`
	UpdatedAt                  string  `json:"updated_at,omitempty"`
}

type UpdateShopSettingsInput struct {
	ShippingFeeLAK             float64
	FreeShippingMinSubtotalLAK float64
	BcelQrEnabled              bool
	CodEnabled                 bool
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
	if err := s.repo.Save(ctx, cfg); err != nil {
		return ShopSettingsView{}, err
	}
	return shopSettingsViewFromModel(cfg), nil
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
		UpdatedAt:                  updatedAt,
	}
}
