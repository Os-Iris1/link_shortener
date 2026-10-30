package services_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"go.uber.org/mock/gomock"

	cerrors "shorter/core/errors"
	"shorter/core/models"
	"shorter/core/ports/mocks"
	"shorter/core/services"
)

func TestLinkServiceShorten(t *testing.T) {
	failure := errors.New("dependency failed")
	tests := []struct {
		name     string
		url      string
		canceled bool
		setup    func(*mocks.MockLinkRepository, *mocks.MockCodeGenerator)
		want     string
		wantErr  error
	}{
		{name: "success", url: "https://example.com/path?q=1#part", setup: func(r *mocks.MockLinkRepository, g *mocks.MockCodeGenerator) {
			gomock.InOrder(
				g.EXPECT().Generate().Return("abc123", nil),
				r.EXPECT().Save(gomock.Any(), models.Link{Code: "abc123", URL: "https://example.com/path?q=1#part"}).Return(nil),
			)
		}, want: "https://short.example/abc123"},
		{name: "http", url: "http://example.com", setup: func(r *mocks.MockLinkRepository, g *mocks.MockCodeGenerator) {
			g.EXPECT().Generate().Return("abc", nil)
			r.EXPECT().Save(gomock.Any(), models.Link{Code: "abc", URL: "http://example.com"}).Return(nil)
		}, want: "https://short.example/abc"},
		{name: "empty", wantErr: cerrors.ErrInvalidURL},
		{name: "relative", url: "/path", wantErr: cerrors.ErrInvalidURL},
		{name: "unsupported scheme", url: "ftp://example.com", wantErr: cerrors.ErrInvalidURL},
		{name: "missing host", url: "https:///path", wantErr: cerrors.ErrInvalidURL},
		{name: "credentials", url: "https://user:pass@example.com", wantErr: cerrors.ErrInvalidURL},
		{name: "bad escape", url: "https://example.com/%zz", wantErr: cerrors.ErrInvalidURL},
		{name: "too long", url: "https://example.com/" + strings.Repeat("a", 8192), wantErr: cerrors.ErrInvalidURL},
		{name: "canceled", url: "https://example.com", canceled: true, wantErr: context.Canceled},
		{name: "generator error", url: "https://example.com", setup: func(r *mocks.MockLinkRepository, g *mocks.MockCodeGenerator) {
			g.EXPECT().Generate().Return("", failure)
		}, wantErr: failure},
		{name: "repository error", url: "https://example.com", setup: func(r *mocks.MockLinkRepository, g *mocks.MockCodeGenerator) {
			g.EXPECT().Generate().Return("abc", nil)
			r.EXPECT().Save(gomock.Any(), models.Link{Code: "abc", URL: "https://example.com"}).Return(failure)
		}, wantErr: failure},
		{name: "collision retry", url: "https://example.com", setup: func(r *mocks.MockLinkRepository, g *mocks.MockCodeGenerator) {
			gomock.InOrder(
				g.EXPECT().Generate().Return("taken", nil),
				r.EXPECT().Save(gomock.Any(), models.Link{Code: "taken", URL: "https://example.com"}).Return(cerrors.ErrCodeConflict),
				g.EXPECT().Generate().Return("free", nil),
				r.EXPECT().Save(gomock.Any(), models.Link{Code: "free", URL: "https://example.com"}).Return(nil),
			)
		}, want: "https://short.example/free"},
		{name: "collision limit", url: "https://example.com", setup: func(r *mocks.MockLinkRepository, g *mocks.MockCodeGenerator) {
			g.EXPECT().Generate().Return("taken", nil).Times(5)
			r.EXPECT().Save(gomock.Any(), models.Link{Code: "taken", URL: "https://example.com"}).Return(cerrors.ErrCodeConflict).Times(5)
		}, wantErr: cerrors.ErrCodeConflict},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			repository := mocks.NewMockLinkRepository(ctrl)
			generator := mocks.NewMockCodeGenerator(ctrl)
			if tt.setup != nil {
				tt.setup(repository, generator)
			}
			service, err := services.NewLinkService(repository, generator, "https://short.example/")
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if tt.canceled {
				cancel()
			}
			got, err := service.Shorten(ctx, tt.url)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("URL = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNewLinkService(t *testing.T) {
	for _, tt := range []struct {
		name, baseURL string
		valid         bool
	}{
		{"valid", "https://short.example", true},
		{"path prefix", "https://example.com/short/", true},
		{"empty", "", false},
		{"relative", "/short", false},
		{"query", "https://example.com/?key=value", false},
		{"empty query", "https://example.com/?", false},
		{"fragment", "https://example.com/#part", false},
		{"empty fragment", "https://example.com/#", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := services.NewLinkService(nil, nil, tt.baseURL)
			if (err == nil) != tt.valid {
				t.Fatalf("error = %v, valid = %v", err, tt.valid)
			}
		})
	}
}
