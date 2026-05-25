package upload

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	PaymentReceiptSubdir = "payment-receipts"
	maxReceiptBytes      = 5 << 20 // 5 MiB
)

var allowedReceiptMIME = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

// PaymentReceiptStore saves payment proof images to disk.
type PaymentReceiptStore struct {
	baseDir   string
	urlPrefix string
}

func NewPaymentReceiptStore(baseDir, urlPrefix string) (*PaymentReceiptStore, error) {
	if baseDir == "" {
		baseDir = "uploads"
	}
	if urlPrefix == "" {
		urlPrefix = "/uploads"
	}
	dir := filepath.Join(baseDir, PaymentReceiptSubdir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create upload dir: %w", err)
	}
	return &PaymentReceiptStore{baseDir: baseDir, urlPrefix: strings.TrimRight(urlPrefix, "/")}, nil
}

// Save stores the uploaded file and returns a public URL path (e.g. /uploads/payment-receipts/abc.jpg).
func (s *PaymentReceiptStore) Save(file *multipart.FileHeader) (string, error) {
	if file == nil {
		return "", fmt.Errorf("payment receipt file is required")
	}
	if file.Size > maxReceiptBytes {
		return "", fmt.Errorf("payment receipt must be at most 5 MB")
	}
	ext, ok := allowedReceiptMIME[file.Header.Get("Content-Type")]
	if !ok {
		// fallback by filename extension
		switch strings.ToLower(filepath.Ext(file.Filename)) {
		case ".jpg", ".jpeg":
			ext = ".jpg"
		case ".png":
			ext = ".png"
		case ".webp":
			ext = ".webp"
		default:
			return "", fmt.Errorf("payment receipt must be JPEG, PNG, or WebP")
		}
	}
	name := fmt.Sprintf("%d_%s%s", time.Now().UnixNano(), randomToken(8), ext)
	dest := filepath.Join(s.baseDir, PaymentReceiptSubdir, name)

	src, err := file.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	out, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return "", err
	}
	defer out.Close()

	written, err := io.Copy(out, io.LimitReader(src, maxReceiptBytes+1))
	if err != nil {
		_ = os.Remove(dest)
		return "", err
	}
	if written > maxReceiptBytes {
		_ = os.Remove(dest)
		return "", fmt.Errorf("payment receipt must be at most 5 MB")
	}

	return fmt.Sprintf("%s/%s/%s", s.urlPrefix, PaymentReceiptSubdir, name), nil
}

func randomToken(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%x", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}
