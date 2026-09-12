package handler

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/mmuly/my-bookmark-be/internal/api"
)

type MetadataHandler struct{}

func NewMetadataHandler() *MetadataHandler {
	return &MetadataHandler{}
}

var (
	reTitle        = regexp.MustCompile(`(?i)<title[^>]*>(.*?)</title>`)
	reMetaDesc     = regexp.MustCompile(`(?i)<meta[^>]+name=["']description["'][^>]+content=["']([^"']*)["']`)
	reMetaDescAlt  = regexp.MustCompile(`(?i)<meta[^>]+content=["']([^"']*)["'][^>]+name=["']description["']`)
	reOGTitle      = regexp.MustCompile(`(?i)<meta[^>]+property=["']og:title["'][^>]+content=["']([^"']*)["']`)
	reOGTitleAlt   = regexp.MustCompile(`(?i)<meta[^>]+content=["']([^"']*)["'][^>]+property=["']og:title["']`)
	reOGDesc       = regexp.MustCompile(`(?i)<meta[^>]+property=["']og:description["'][^>]+content=["']([^"']*)["']`)
	reOGDescAlt    = regexp.MustCompile(`(?i)<meta[^>]+content=["']([^"']*)["'][^>]+property=["']og:description["']`)
	reOGImage      = regexp.MustCompile(`(?i)<meta[^>]+property=["']og:image["'][^>]+content=["']([^"']*)["']`)
	reOGImageAlt   = regexp.MustCompile(`(?i)<meta[^>]+content=["']([^"']*)["'][^>]+property=["']og:image["']`)
	reAppleIcon    = regexp.MustCompile(`(?i)<link[^>]+rel=["']apple-touch-icon["'][^>]+href=["']([^"']*)["']`)
	reAppleIconAlt = regexp.MustCompile(`(?i)<link[^>]+href=["']([^"']*)["'][^>]+rel=["']apple-touch-icon["']`)
	reIcon         = regexp.MustCompile(`(?i)<link[^>]+rel=["'](?:shortcut )?icon["'][^>]+href=["']([^"']*)["']`)
	reIconAlt      = regexp.MustCompile(`(?i)<link[^>]+href=["']([^"']*)["'][^>]+rel=["'](?:shortcut )?icon["']`)
)

func matchFirst(body string, re1, re2 *regexp.Regexp) string {
	if m := re1.FindStringSubmatch(body); len(m) > 1 {
		return strings.TrimSpace(m[1])
	}
	if re2 != nil {
		if m := re2.FindStringSubmatch(body); len(m) > 1 {
			return strings.TrimSpace(m[1])
		}
	}
	return ""
}

func resolveURL(base, href string) string {
	if href == "" {
		return ""
	}
	if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
		return href
	}
	u, err := url.Parse(base)
	if err != nil {
		return href
	}
	ref, err := url.Parse(href)
	if err != nil {
		return href
	}
	return u.ResolveReference(ref).String()
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func (h *MetadataHandler) FetchMetadata(ctx context.Context, request api.FetchMetadataRequestObject) (api.FetchMetadataResponseObject, error) {
	if request.Body == nil || strings.TrimSpace(request.Body.Url) == "" {
		return api.FetchMetadata400JSONResponse{Message: "URL is required"}, nil
	}
	targetURL := request.Body.Url

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return api.FetchMetadata400JSONResponse{Message: "invalid URL"}, nil
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; MyBookmark/1.0)")

	resp, err := client.Do(req)
	if err != nil {
		return api.FetchMetadata400JSONResponse{Message: fmt.Sprintf("failed to fetch URL: %v", err)}, nil
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 512*1024)) // max 512KB
	if err != nil {
		return nil, err
	}
	body := string(bodyBytes)

	title := matchFirst(body, reOGTitle, reOGTitleAlt)
	if title == "" {
		title = matchFirst(body, reTitle, nil)
	}
	if title == "" {
		title = targetURL
	}

	description := matchFirst(body, reOGDesc, reOGDescAlt)
	if description == "" {
		description = matchFirst(body, reMetaDesc, reMetaDescAlt)
	}

	imageURL := matchFirst(body, reOGImage, reOGImageAlt)
	if imageURL != "" {
		imageURL = resolveURL(targetURL, imageURL)
	}

	favicon := matchFirst(body, reAppleIcon, reAppleIconAlt)
	if favicon == "" {
		favicon = matchFirst(body, reIcon, reIconAlt)
	}
	if favicon != "" {
		favicon = resolveURL(targetURL, favicon)
	}
	if favicon == "" {
		if u, err := url.Parse(targetURL); err == nil {
			favicon = fmt.Sprintf("%s://%s/favicon.ico", u.Scheme, u.Host)
		}
	}

	return api.FetchMetadata200JSONResponse{
		Title:       title,
		Description: strPtr(description),
		Favicon:     strPtr(favicon),
		ImageUrl:    strPtr(imageURL),
	}, nil
}
