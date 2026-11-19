package services

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	cerrors "shorter/core/errors"
	"shorter/core/models"
	"shorter/core/ports"
)

const (
	maxAttempts = 5
	decimalBase = 10
	portBitSize = 16
)

type LinkService struct {
	repository ports.LinkRepository
	generator  ports.CodeGenerator
	baseURL    string
}

func NewLinkService(repository ports.LinkRepository, generator ports.CodeGenerator, baseURL string) (*LinkService, error) {
	u, err := parseURL(baseURL)
	if err != nil || u.RawQuery != "" || u.ForceQuery || strings.Contains(baseURL, "#") {
		return nil, fmt.Errorf("invalid BASE_URL: %w", cerrors.ErrInvalidURL)
	}
	return &LinkService{repository: repository, generator: generator, baseURL: strings.TrimRight(baseURL, "/")}, nil
}

func (s *LinkService) Shorten(ctx context.Context, originalURL string) (string, error) {
	if _, err := parseURL(originalURL); err != nil {
		return "", cerrors.ErrInvalidURL
	}
	for range maxAttempts {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		code, err := s.generator.Generate()
		if err != nil {
			return "", fmt.Errorf("generate short code: %w", err)
		}
		err = s.repository.Save(ctx, models.Link{Code: code, URL: originalURL})
		if errors.Is(err, cerrors.ErrCodeConflict) {
			continue
		}
		if err != nil {
			return "", fmt.Errorf("save link: %w", err)
		}
		return s.baseURL + "/" + code, nil
	}
	return "", fmt.Errorf("short code attempts exhausted: %w", cerrors.ErrCodeConflict)
}

func parseURL(value string) (*url.URL, error) {
	if len(value) == 0 || len(value) > models.MaxURLBytes {
		return nil, cerrors.ErrInvalidURL
	}
	u, err := url.Parse(value)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.User != nil {
		return nil, cerrors.ErrInvalidURL
	}
	if port := u.Port(); port != "" {
		number, err := strconv.ParseUint(port, decimalBase, portBitSize)
		if err != nil || number == 0 {
			return nil, cerrors.ErrInvalidURL
		}
	} else if strings.HasSuffix(u.Host, ":") {
		return nil, cerrors.ErrInvalidURL
	}
	return u, nil
}

func (s *LinkService) Resolve(ctx context.Context, code string) (string, error) {
	if len(code) != models.CodeLength {
		return "", cerrors.ErrInvalidCode
	}
	for _, char := range code {
		valid := char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' || char >= '0' && char <= '9' || char == '-' || char == '_'
		if !valid {
			return "", cerrors.ErrInvalidCode
		}
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	link, err := s.repository.FindByCode(ctx, code)
	if err != nil {
		return "", fmt.Errorf("find link: %w", err)
	}
	return link.URL, nil
}
