package service

import (
	"errors"
	"sync"
	"time"
)

// OTPService stores one-time codes in memory (stub until SMS provider is wired).
type OTPService struct {
	mu       sync.Mutex
	codes    map[string]otpEntry
	stubCode string
	ttl      time.Duration
}

type otpEntry struct {
	Code      string
	ExpiresAt time.Time
}

func NewOTPService(stubCode string, ttlMinutes int) *OTPService {
	if stubCode == "" {
		stubCode = "1234"
	}
	if ttlMinutes <= 0 {
		ttlMinutes = 10
	}
	return &OTPService{
		codes:    make(map[string]otpEntry),
		stubCode: stubCode,
		ttl:      time.Duration(ttlMinutes) * time.Minute,
	}
}

var ErrInvalidPhone = errors.New("invalid phone number")
var ErrOTPNotFound = errors.New("otp not sent or expired")
var ErrOTPInvalid = errors.New("invalid otp code")

// Send stores an OTP for [phone]. SMS integration goes here later.
func (s *OTPService) Send(phone string) error {
	p := NormalizePhone(phone)
	if len(p) < 8 {
		return ErrInvalidPhone
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.codes[p] = otpEntry{
		Code:      s.stubCode,
		ExpiresAt: time.Now().Add(s.ttl),
	}
	return nil
}

// Verify checks the OTP for [phone]. Returns nil when valid.
func (s *OTPService) Verify(phone, code string) error {
	p := NormalizePhone(phone)
	c := NormalizePhone(code)
	if len(p) < 8 {
		return ErrInvalidPhone
	}
	if c == "" {
		return ErrOTPInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.codes[p]
	if !ok || time.Now().After(entry.ExpiresAt) {
		delete(s.codes, p)
		return ErrOTPNotFound
	}
	if entry.Code != c {
		return ErrOTPInvalid
	}
	delete(s.codes, p)
	return nil
}
