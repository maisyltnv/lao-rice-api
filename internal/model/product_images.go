package model

import (
	"encoding/json"
	"strings"
)

const maxProductImages = 12

// ParseProductImageURLs reads stored JSON and falls back to legacy image_url.
func ParseProductImageURLs(jsonRaw, legacyURL string) []string {
	legacyURL = strings.TrimSpace(legacyURL)
	var urls []string
	if raw := strings.TrimSpace(jsonRaw); raw != "" {
		_ = json.Unmarshal([]byte(raw), &urls)
	}
	return NormalizeProductImageURLs(urls, legacyURL)
}

func NormalizeProductImageURLs(urls []string, legacyURL string) []string {
	seen := make(map[string]struct{}, len(urls)+1)
	out := make([]string, 0, len(urls)+1)
	add := func(u string) {
		u = strings.TrimSpace(u)
		if u == "" {
			return
		}
		if _, ok := seen[u]; ok {
			return
		}
		seen[u] = struct{}{}
		out = append(out, u)
	}
	for _, u := range urls {
		add(u)
		if len(out) >= maxProductImages {
			return out
		}
	}
	if len(out) == 0 {
		add(legacyURL)
	}
	return out
}

func MarshalProductImageURLs(urls []string) string {
	if len(urls) == 0 {
		return ""
	}
	raw, err := json.Marshal(urls)
	if err != nil {
		return ""
	}
	return string(raw)
}

func RemovedProductImageURLs(oldURLs, newURLs []string) []string {
	newSet := make(map[string]struct{}, len(newURLs))
	for _, u := range newURLs {
		u = strings.TrimSpace(u)
		if u != "" {
			newSet[u] = struct{}{}
		}
	}
	var removed []string
	for _, u := range oldURLs {
		u = strings.TrimSpace(u)
		if u == "" {
			continue
		}
		if _, ok := newSet[u]; !ok {
			removed = append(removed, u)
		}
	}
	return removed
}
