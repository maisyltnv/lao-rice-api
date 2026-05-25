package service

import (
	"context"
	"errors"
	"strings"

	"shopapi/internal/model"
	"shopapi/internal/repository"

	"gorm.io/gorm"
)

type BannerService struct {
	repo *repository.BannerRepository
}

func NewBannerService(repo *repository.BannerRepository) *BannerService {
	return &BannerService{repo: repo}
}

type CreateBannerInput struct {
	Title       string
	Subtitle    string
	Description string
	ImageURL    string
	CTALabel    string
	LinkURL     string
	SortOrder   int
	IsActive    *bool
}

type UpdateBannerInput struct {
	Title       *string
	Subtitle    *string
	Description *string
	ImageURL    *string
	CTALabel    *string
	LinkURL     *string
	SortOrder   *int
	IsActive    *bool
}

func (s *BannerService) ListPublic(ctx context.Context) ([]model.Banner, error) {
	return s.repo.List(ctx, true)
}

func (s *BannerService) ListAll(ctx context.Context) ([]model.Banner, error) {
	return s.repo.List(ctx, false)
}

func (s *BannerService) GetPublic(ctx context.Context, id uint64) (*model.Banner, error) {
	b, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if !b.IsActive {
		return nil, gorm.ErrRecordNotFound
	}
	return b, nil
}

func (s *BannerService) GetByID(ctx context.Context, id uint64) (*model.Banner, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *BannerService) Create(ctx context.Context, in CreateBannerInput) (*model.Banner, error) {
	if err := validateBannerFields(in.Title, in.ImageURL); err != nil {
		return nil, err
	}
	active := true
	if in.IsActive != nil {
		active = *in.IsActive
	}
	b := &model.Banner{
		Title:       strings.TrimSpace(in.Title),
		Subtitle:    strings.TrimSpace(in.Subtitle),
		Description: strings.TrimSpace(in.Description),
		ImageURL:    strings.TrimSpace(in.ImageURL),
		CTALabel:    strings.TrimSpace(in.CTALabel),
		LinkURL:     strings.TrimSpace(in.LinkURL),
		SortOrder:   in.SortOrder,
		IsActive:    active,
	}
	if err := s.repo.Create(ctx, b); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, b.ID)
}

func (s *BannerService) Update(ctx context.Context, id uint64, in UpdateBannerInput) (*model.Banner, error) {
	b, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if in.Title != nil {
		if strings.TrimSpace(*in.Title) == "" {
			return nil, errors.New("title is required")
		}
		b.Title = strings.TrimSpace(*in.Title)
	}
	if in.Subtitle != nil {
		b.Subtitle = strings.TrimSpace(*in.Subtitle)
	}
	if in.Description != nil {
		b.Description = strings.TrimSpace(*in.Description)
	}
	if in.ImageURL != nil {
		if strings.TrimSpace(*in.ImageURL) == "" {
			return nil, errors.New("image_url is required")
		}
		b.ImageURL = strings.TrimSpace(*in.ImageURL)
	}
	if in.CTALabel != nil {
		b.CTALabel = strings.TrimSpace(*in.CTALabel)
	}
	if in.LinkURL != nil {
		b.LinkURL = strings.TrimSpace(*in.LinkURL)
	}
	if in.SortOrder != nil {
		b.SortOrder = *in.SortOrder
	}
	if in.IsActive != nil {
		b.IsActive = *in.IsActive
	}
	if err := s.repo.Update(ctx, b); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, id)
}

func (s *BannerService) Delete(ctx context.Context, id uint64) error {
	if _, err := s.repo.GetByID(ctx, id); err != nil {
		return err
	}
	return s.repo.Delete(ctx, id)
}

func validateBannerFields(title, imageURL string) error {
	if strings.TrimSpace(title) == "" {
		return errors.New("title is required")
	}
	if strings.TrimSpace(imageURL) == "" {
		return errors.New("image_url is required")
	}
	return nil
}
