package service

import (
	"context"
	"errors"
	"math"

	"shopapi/internal/model"
	"shopapi/internal/repository"

	"gorm.io/gorm"
)

// CalculateFinalPriceLAK computes retail price in LAK from CNY cost, FX, and margin.
// Formula: FinalPriceLAK = (OriginalPriceCNY * ExchangeRate) * (1 + ProfitMargin).
func CalculateFinalPriceLAK(originalPriceCNY, exchangeRate, profitMargin float64) float64 {
	raw := (originalPriceCNY * exchangeRate) * (1 + profitMargin)
	// Round to 2 decimal places for currency display consistency.
	return math.Round(raw*100) / 100
}

type ProductService struct {
	repo     *repository.ProductRepository
	category *repository.CategoryRepository
}

func NewProductService(repo *repository.ProductRepository, category *repository.CategoryRepository) *ProductService {
	return &ProductService{repo: repo, category: category}
}

func (s *ProductService) validateCategory(ctx context.Context, categoryID *uint64) error {
	if categoryID == nil {
		return errors.New("category_id is required")
	}
	if *categoryID == 0 {
		return errors.New("invalid category_id")
	}
	cat, err := s.category.GetByID(ctx, *categoryID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("category not found")
		}
		return err
	}
	if !cat.IsActive {
		return errors.New("category is not active")
	}
	return nil
}

type CreateProductInput struct {
	Name             string
	Description      string
	ImageURL         string
	CategoryID       *uint64
	OriginalPriceCNY float64
	ExchangeRate     float64
	ProfitMargin     float64
	FinalPriceLAK    *float64
	Stock            int
	SourceURL        string
}

type UpdateProductInput struct {
	Name             *string
	Description      *string
	ImageURL         *string
	ClearCategory    bool
	CategoryID       *uint64
	OriginalPriceCNY *float64
	ExchangeRate     *float64
	ProfitMargin     *float64
	FinalPriceLAK    *float64
	Stock            *int
	SourceURL        *string
}

func (s *ProductService) Create(ctx context.Context, in CreateProductInput) (*model.Product, error) {
	if err := s.validateCategory(ctx, in.CategoryID); err != nil {
		return nil, err
	}
	if in.Stock < 0 {
		return nil, errors.New("stock must be >= 0")
	}
	if in.OriginalPriceCNY < 0 || in.ExchangeRate < 0 || in.ProfitMargin < -1 {
		return nil, errors.New("invalid pricing inputs")
	}
	final := CalculateFinalPriceLAK(in.OriginalPriceCNY, in.ExchangeRate, in.ProfitMargin)
	if in.FinalPriceLAK != nil {
		final = *in.FinalPriceLAK
	}
	p := &model.Product{
		Name:             in.Name,
		Description:      in.Description,
		ImageURL:         in.ImageURL,
		CategoryID:       in.CategoryID,
		OriginalPriceCNY: in.OriginalPriceCNY,
		ExchangeRate:     in.ExchangeRate,
		ProfitMargin:     in.ProfitMargin,
		FinalPriceLAK:    final,
		Stock:            in.Stock,
		SourceURL:        in.SourceURL,
	}
	if err := s.repo.Create(ctx, p); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, p.ID)
}

func (s *ProductService) GetByID(ctx context.Context, id uint64) (*model.Product, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *ProductService) List(ctx context.Context, limit, offset int, categoryID *uint64, search string) ([]model.Product, int64, error) {
	if limit <= 0 || limit > 500 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.List(ctx, limit, offset, categoryID, search)
}

func (s *ProductService) Update(ctx context.Context, id uint64, in UpdateProductInput) (*model.Product, error) {
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		return nil, err
	}
	if in.ClearCategory {
		p.CategoryID = nil
		p.Category = nil
	} else if in.CategoryID != nil {
		if err := s.validateCategory(ctx, in.CategoryID); err != nil {
			return nil, err
		}
		p.CategoryID = in.CategoryID
		p.Category = nil // drop stale preload before Save
	}
	if in.Name != nil {
		p.Name = *in.Name
	}
	if in.Description != nil {
		p.Description = *in.Description
	}
	if in.ImageURL != nil {
		p.ImageURL = *in.ImageURL
	}
	if in.OriginalPriceCNY != nil {
		p.OriginalPriceCNY = *in.OriginalPriceCNY
	}
	if in.ExchangeRate != nil {
		p.ExchangeRate = *in.ExchangeRate
	}
	if in.ProfitMargin != nil {
		p.ProfitMargin = *in.ProfitMargin
	}
	if in.Stock != nil {
		if *in.Stock < 0 {
			return nil, errors.New("stock must be >= 0")
		}
		p.Stock = *in.Stock
	}
	if in.SourceURL != nil {
		p.SourceURL = *in.SourceURL
	}
	if in.FinalPriceLAK != nil {
		p.FinalPriceLAK = *in.FinalPriceLAK
	} else if in.OriginalPriceCNY != nil || in.ExchangeRate != nil || in.ProfitMargin != nil {
		if p.OriginalPriceCNY < 0 || p.ExchangeRate < 0 || p.ProfitMargin < -1 {
			return nil, errors.New("invalid pricing inputs")
		}
		p.FinalPriceLAK = CalculateFinalPriceLAK(p.OriginalPriceCNY, p.ExchangeRate, p.ProfitMargin)
	}
	if err := s.repo.Update(ctx, p); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, id)
}

func (s *ProductService) Delete(ctx context.Context, id uint64) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	return nil
}
