package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"shopapi/internal/delivery"
	"shopapi/internal/model"
	"shopapi/internal/repository"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthService struct {
	users   *repository.UserRepository
	secret  []byte
	expiryH int
}

func NewAuthService(users *repository.UserRepository, jwtSecret string, expiryHours int) *AuthService {
	if expiryHours <= 0 {
		expiryHours = 72
	}
	return &AuthService{
		users:   users,
		secret:  []byte(jwtSecret),
		expiryH: expiryHours,
	}
}

type RegisterInput struct {
	Username string
	Password string
}

// Register creates a storefront customer account (role is always user).
func (s *AuthService) Register(ctx context.Context, in RegisterInput) (*model.User, error) {
	if len(in.Password) < 8 {
		return nil, errors.New("password too short")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	u := &model.User{
		Username:     in.Username,
		PasswordHash: string(hash),
		Role:         model.RoleUser,
	}
	if err := s.users.Create(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

type RegisterAdminInput struct {
	Username string
	Password string
}

// RegisterAdmin creates an admin user (no shared secret — do not expose this endpoint on the public internet).
func (s *AuthService) RegisterAdmin(ctx context.Context, in RegisterAdminInput) (*model.User, error) {
	if len(in.Password) < 8 {
		return nil, errors.New("password too short")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	u := &model.User{
		Username:     in.Username,
		PasswordHash: string(hash),
		Role:         model.RoleAdmin,
	}
	if err := s.users.Create(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

type LoginInput struct {
	Username string
	Password string
}

type TokenPair struct {
	Token     string
	ExpiresAt time.Time
}

// Login issues a JWT for any valid user (customer or admin).
func (s *AuthService) Login(ctx context.Context, in LoginInput) (*model.User, *TokenPair, error) {
	return s.login(ctx, in, "")
}

// LoginAdmin issues a JWT only if the user exists and has admin role.
func (s *AuthService) LoginAdmin(ctx context.Context, in LoginInput) (*model.User, *TokenPair, error) {
	return s.login(ctx, in, model.RoleAdmin)
}

func (s *AuthService) login(ctx context.Context, in LoginInput, requireRole string) (*model.User, *TokenPair, error) {
	u, err := s.users.GetByUsername(ctx, in.Username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, errors.New("invalid credentials")
		}
		return nil, nil, err
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(in.Password)) != nil {
		return nil, nil, errors.New("invalid credentials")
	}
	if requireRole != "" && u.Role != requireRole {
		return nil, nil, errors.New("invalid credentials")
	}
	exp := time.Now().Add(time.Duration(s.expiryH) * time.Hour)
	tok, err := s.signJWT(u.ID, u.Username, u.Role, exp)
	if err != nil {
		return nil, nil, err
	}
	return u, &TokenPair{Token: tok, ExpiresAt: exp}, nil
}

type jwtClaims struct {
	UserID   uint64 `json:"uid"`
	Username string `json:"sub"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

func (s *AuthService) signJWT(userID uint64, username, role string, exp time.Time) (string, error) {
	claims := jwtClaims{
		UserID:   userID,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(exp),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString(s.secret)
}

// FindOrCreatePhoneUser returns a customer account keyed by normalized phone (stored as username).
func (s *AuthService) FindOrCreatePhoneUser(ctx context.Context, phone string) (*model.User, error) {
	p := NormalizePhone(phone)
	if len(p) < 8 {
		return nil, ErrInvalidPhone
	}
	u, err := s.users.GetByUsername(ctx, p)
	if err == nil {
		return u, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	secret := make([]byte, 16)
	if _, err := rand.Read(secret); err != nil {
		return nil, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(hex.EncodeToString(secret)), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	u = &model.User{
		Username:     p,
		PasswordHash: string(hash),
		Role:         model.RoleUser,
	}
	if err := s.users.Create(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

// IssueTokenForUser signs a JWT for an existing user.
func (s *AuthService) IssueTokenForUser(u *model.User) (*TokenPair, error) {
	exp := time.Now().Add(time.Duration(s.expiryH) * time.Hour)
	tok, err := s.signJWT(u.ID, u.Username, u.Role, exp)
	if err != nil {
		return nil, err
	}
	return &TokenPair{Token: tok, ExpiresAt: exp}, nil
}

func (s *AuthService) ParseToken(token string) (*jwtClaims, error) {
	parsed, err := jwt.ParseWithClaims(token, &jwtClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return s.secret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := parsed.Claims.(*jwtClaims)
	if !ok || !parsed.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

// GetUserByID loads a user row (including saved shipping profile).
func (s *AuthService) GetUserByID(ctx context.Context, id uint64) (*model.User, error) {
	return s.users.GetByID(ctx, id)
}

// DeleteAccount permanently deletes the user's account and personal profile data.
func (s *AuthService) DeleteAccount(ctx context.Context, id uint64) error {
	return s.users.Delete(ctx, id)
}

// UpdateProfileInput is the customer default shipping address for checkout.
type UpdateProfileInput struct {
	RecipientName     string
	ShippingPhone     string
	Province          string
	AddressDetail     string
	DeliveryLatitude  float64
	DeliveryLongitude float64
}

// UpdateCustomerProfile saves default shipping fields on the user account.
func (s *AuthService) UpdateCustomerProfile(ctx context.Context, userID uint64, in UpdateProfileInput) (*model.User, error) {
	u, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if u.Role != model.RoleUser {
		return nil, errors.New("only customer accounts have a shipping profile")
	}
	name := strings.TrimSpace(in.RecipientName)
	if name == "" {
		return nil, errors.New("recipient_name is required")
	}
	phone := strings.TrimSpace(in.ShippingPhone)
	if phone == "" {
		phone = u.Username
	}
	phone = NormalizePhone(phone)
	if len(phone) < 8 {
		return nil, ErrInvalidPhone
	}
	addr := strings.TrimSpace(in.AddressDetail)
	if addr == "" {
		return nil, errors.New("address_detail is required")
	}
	province := strings.TrimSpace(in.Province)
	if province == "" {
		province = "ນະຄອນຫຼວງວຽງຈັນ"
	}
	lat, lng := in.DeliveryLatitude, in.DeliveryLongitude
	if lat != 0 || lng != 0 {
		if err := delivery.ValidateCoordinates(lat, lng); err != nil {
			return nil, err
		}
	}
	u.RecipientName = name
	u.ShippingPhone = phone
	u.Province = province
	u.AddressDetail = addr
	u.DeliveryLatitude = lat
	u.DeliveryLongitude = lng
	if err := s.users.Save(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}
