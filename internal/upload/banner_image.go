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
	BannerImageSubdir   = "banner-images"
	maxBannerImageBytes = 5 << 20 // 5 MiB
)

var allowedBannerImageMIME = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

// BannerImageStore saves admin hero banner photos to disk.
type BannerImageStore struct {
	baseDir   string
	urlPrefix string
}

func NewBannerImageStore(baseDir, urlPrefix string) (*BannerImageStore, error) {
	if baseDir == "" {
		baseDir = "uploads"
	}
	if urlPrefix == "" {
		urlPrefix = "/uploads"
	}
	dir := filepath.Join(baseDir, BannerImageSubdir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create banner image dir: %w", err)
	}
	return &BannerImageStore{
		baseDir:   baseDir,
		urlPrefix: strings.TrimRight(urlPrefix, "/"),
	}, nil
}

func (s *BannerImageStore) IsManagedURL(url string) bool {
	rel, ok := s.relativePath(url)
	return ok && rel != ""
}

func (s *BannerImageStore) relativePath(url string) (string, bool) {
	trimmed := strings.TrimSpace(url)
	if trimmed == "" {
		return "", false
	}
	prefix := s.urlPrefix + "/" + BannerImageSubdir + "/"
	var pathPart string
	switch {
	case strings.HasPrefix(trimmed, prefix):
		pathPart = strings.TrimPrefix(trimmed, prefix)
	case strings.Contains(trimmed, "://"):
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

func (s *BannerImageStore) Save(file *multipart.FileHeader) (string, error) {
	if file == nil {
		return "", fmt.Errorf("image file is required")
	}
	if file.Size <= 0 {
		return "", fmt.Errorf("image file is empty")
	}
	if file.Size > maxBannerImageBytes {
		return "", fmt.Errorf("image must be at most 5 MB")
	}

	src, err := file.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	ext, head, extErr := bannerImageExt(file.Header.Get("Content-Type"), file.Filename, src)
	if extErr != nil {
		return "", extErr
	}

	name := fmt.Sprintf("%d_%s%s", time.Now().UnixNano(), randomToken(8), ext)
	dest := filepath.Join(s.baseDir, BannerImageSubdir, name)

	out, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return "", err
	}
	defer out.Close()

	reader := io.MultiReader(bytes.NewReader(head), src)
	written, err := io.Copy(out, io.LimitReader(reader, maxBannerImageBytes+1))
	if err != nil {
		_ = os.Remove(dest)
		return "", err
	}
	if written > maxBannerImageBytes {
		_ = os.Remove(dest)
		return "", fmt.Errorf("image must be at most 5 MB")
	}

	return fmt.Sprintf("%s/%s/%s", s.urlPrefix, BannerImageSubdir, name), nil
}

func (s *BannerImageStore) DeleteByURL(url string) error {
	rel, ok := s.relativePath(url)
	if !ok {
		return nil
	}
	full := filepath.Join(s.baseDir, BannerImageSubdir, rel)
	cleanBase := filepath.Clean(filepath.Join(s.baseDir, BannerImageSubdir))
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

func bannerImageExt(contentType, filename string, src multipart.File) (string, []byte, error) {
	if ext, ok := allowedBannerImageMIME[strings.ToLower(strings.TrimSpace(contentType))]; ok {
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
