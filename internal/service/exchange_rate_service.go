package service

import (
	"context"
	"errors"

	"shopapi/internal/repository"

	"gorm.io/gorm"
)

type ExchangeRateService struct {
	rates    *repository.ExchangeRateRepository
	products *repository.ProductRepository
}

func NewExchangeRateService(rates *repository.ExchangeRateRepository, products *repository.ProductRepository) *ExchangeRateService {
	return &ExchangeRateService{rates: rates, products: products}
}

type ExchangeRateView struct {
	RateLAKPerCNY float64               `json:"rate_lak_per_cny"`
	ProductsCount int64                 `json:"products_count"`
	UpdatedAt     string                `json:"updated_at,omitempty"`
	Examples      []ExchangeRateExample `json:"examples,omitempty"`
}

type ExchangeRateExample struct {
	CNY float64 `json:"cny"`
	LAK float64 `json:"lak"`
}

var previewCNYAmounts = []float64{10, 50, 100, 500}

func (s *ExchangeRateService) buildExamples(rate float64) []ExchangeRateExample {
	out := make([]ExchangeRateExample, 0, len(previewCNYAmounts))
	for _, cny := range previewCNYAmounts {
		out = append(out, ExchangeRateExample{
			CNY: cny,
			LAK: CalculateFinalPriceLAK(cny, rate, 0),
		})
	}
	return out
}

// Get returns the current shop-wide rate and how many products exist.
func (s *ExchangeRateService) Get(ctx context.Context) (*ExchangeRateView, error) {
	cfg, err := s.rates.GetOrCreate(ctx)
	if err != nil {
		return nil, err
	}
	count, err := s.products.CountAll(ctx)
	if err != nil {
		return nil, err
	}
	return &ExchangeRateView{
		RateLAKPerCNY: cfg.RateLAK,
		ProductsCount: count,
		UpdatedAt:     cfg.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		Examples:      s.buildExamples(cfg.RateLAK),
	}, nil
}

type SetExchangeRateResult struct {
	RateLAKPerCNY   float64 `json:"rate_lak_per_cny"`
	ProductsUpdated int64   `json:"products_updated"`
	UpdatedAt       string  `json:"updated_at"`
}

// Set updates the global rate and recalculates final_price_lak for every product.
// Formula per product: final_price_lak = (original_price_cny * rate) * (1 + profit_margin).
func (s *ExchangeRateService) Set(ctx context.Context, rate float64) (*SetExchangeRateResult, error) {
	if rate <= 0 {
		return nil, errors.New("rate must be greater than 0")
	}
	cfg, err := s.rates.GetOrCreate(ctx)
	if err != nil {
		return nil, err
	}
	cfg.RateLAK = rate

	products, err := s.products.ListAllForPricing(ctx)
	if err != nil {
		return nil, err
	}

	err = s.rates.Transaction(ctx, func(tx *gorm.DB) error {
		if err := tx.Save(cfg).Error; err != nil {
			return err
		}
		for i := range products {
			p := &products[i]
			p.ExchangeRate = rate
			p.FinalPriceLAK = CalculateFinalPriceLAK(p.OriginalPriceCNY, rate, p.ProfitMargin)
			if err := tx.Model(p).Updates(map[string]interface{}{
				"exchange_rate":   p.ExchangeRate,
				"final_price_lak": p.FinalPriceLAK,
			}).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Reload config for updated_at after save
	cfg, _ = s.rates.GetOrCreate(ctx)

	return &SetExchangeRateResult{
		RateLAKPerCNY:   rate,
		ProductsUpdated: int64(len(products)),
		UpdatedAt:       cfg.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}
