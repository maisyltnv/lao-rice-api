package upload

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	ProductImageSubdir  = "product-images"
	maxProductImageBytes = 5 << 20 // 5 MiB
)

var allowedProductImageMIME = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

// ProductImageStore saves admin product photos to disk.
type ProductImageStore struct {
	baseDir   string
	urlPrefix string
}

func NewProductImageStore(baseDir, urlPrefix string) (*ProductImageStore, error) {
	if baseDir == "" {
		baseDir = "uploads"
	}
	if urlPrefix == "" {
		urlPrefix = "/uploads"
	}
	dir := filepath.Join(baseDir, ProductImageSubdir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create product image dir: %w", err)
	}
	return &ProductImageStore{
		baseDir:   baseDir,
		urlPrefix: strings.TrimRight(urlPrefix, "/"),
	}, nil
}

// IsManagedURL reports whether url points to a file owned by this store.
func (s *ProductImageStore) IsManagedURL(url string) bool {
	rel, ok := s.relativePath(url)
	return ok && rel != ""
}

func (s *ProductImageStore) relativePath(url string) (string, bool) {
	trimmed := strings.TrimSpace(url)
	if trimmed == "" {
		return "", false
	}
	prefix := s.urlPrefix + "/" + ProductImageSubdir + "/"
	var pathPart string
	switch {
	case strings.HasPrefix(trimmed, prefix):
		pathPart = strings.TrimPrefix(trimmed, prefix)
	case strings.Contains(trimmed, "://"):
		// Absolute URL — only manage our upload path segment.
		idx := strings.Index(trimmed, prefix)
		if idx < 0 {
			return "", false
		}
		pathPart = strings.TrimPrefix(trimmed[idx:], prefix)
	default:
		return "", false
	}
	pathPart = strings.Trim(pathPart, "/")
	if pathPart == "" || strings.Contains(pathPart, "..") {
		return "", false
	}
	return pathPart, true
}

// Save stores the uploaded file and returns a public URL path (e.g. /uploads/product-images/abc.jpg).
func (s *ProductImageStore) Save(file *multipart.FileHeader) (string, error) {
	if file == nil {
		return "", fmt.Errorf("image file is required")
	}
	if file.Size <= 0 {
		return "", fmt.Errorf("image file is empty")
	}
	if file.Size > maxProductImageBytes {
		return "", fmt.Errorf("image must be at most 5 MB")
	}

	src, err := file.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	ext, head, extErr := productImageExt(file.Header.Get("Content-Type"), file.Filename, src)
	if extErr != nil {
		return "", extErr
	}

	name := fmt.Sprintf("%d_%s%s", time.Now().UnixNano(), randomToken(8), ext)
	dest := filepath.Join(s.baseDir, ProductImageSubdir, name)

	out, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return "", err
	}
	defer out.Close()

	reader := io.MultiReader(bytes.NewReader(head), src)
	written, err := io.Copy(out, io.LimitReader(reader, maxProductImageBytes+1))
	if err != nil {
		_ = os.Remove(dest)
		return "", err
	}
	if written > maxProductImageBytes {
		_ = os.Remove(dest)
		return "", fmt.Errorf("image must be at most 5 MB")
	}

	return fmt.Sprintf("%s/%s/%s", s.urlPrefix, ProductImageSubdir, name), nil
}

// DeleteByURL removes a previously uploaded product image. No-op for non-managed URLs.
func (s *ProductImageStore) DeleteByURL(url string) error {
	rel, ok := s.relativePath(url)
	if !ok {
		return nil
	}
	full := filepath.Join(s.baseDir, ProductImageSubdir, rel)
	cleanBase := filepath.Clean(filepath.Join(s.baseDir, ProductImageSubdir))
	cleanFull := filepath.Clean(full)
	if !strings.HasPrefix(cleanFull, cleanBase+string(os.PathSeparator)) && cleanFull != cleanBase {
		return fmt.Errorf("invalid image path")
	}
	if err := os.Remove(cleanFull); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return nil
}

func productImageExt(contentType, filename string, src multipart.File) (string, []byte, error) {
	if ext, ok := allowedProductImageMIME[strings.ToLower(strings.TrimSpace(contentType))]; ok {
		return ext, nil, nil
	}
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".jpg", ".jpeg":
		return ".jpg", nil, nil
	case ".png":
		return ".png", nil, nil
	case ".webp":
		return ".webp", nil, nil
	}

	head := make([]byte, 12)
	n, _ := io.ReadFull(src, head)
	sniff := head[:n]
	if n >= 3 && sniff[0] == 0xff && sniff[1] == 0xd8 && sniff[2] == 0xff {
		return ".jpg", sniff, nil
	}
	if n >= 8 && string(sniff[:8]) == "\x89PNG\r\n\x1a\n" {
		return ".png", sniff, nil
	}
	if n >= 12 && string(sniff[0:4]) == "RIFF" && string(sniff[8:12]) == "WEBP" {
		return ".webp", sniff, nil
	}
	return "", nil, fmt.Errorf("image must be JPEG, PNG, or WebP")
}
