package services_test

import (
	"context"
	"encoding/base64"
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
	const prefix = "https://example.com/"
	longURL := prefix + strings.Repeat("я", (8192-len(prefix))/len("я"))
	longURL += strings.Repeat("a", 8192-len(longURL))
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
		{name: "URL at byte limit", url: longURL, setup: func(r *mocks.MockLinkRepository, g *mocks.MockCodeGenerator) {
			g.EXPECT().Generate().Return("abcdefghijkl", nil)
			r.EXPECT().Save(gomock.Any(), models.Link{Code: "abcdefghijkl", URL: longURL}).Return(nil)
		}, want: "https://short.example/abcdefghijkl"},
		{name: "URL one byte over limit", url: longURL + "a", wantErr: cerrors.ErrInvalidURL},
		{name: "canceled", url: "https://example.com", canceled: true, wantErr: context.Canceled},
		{name: "generator error", url: "https://example.com", setup: func(_ *mocks.MockLinkRepository, g *mocks.MockCodeGenerator) {
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

func TestURLPorts(t *testing.T) {
	for _, tc := range []struct {
		name, host string
		valid      bool
	}{
		{"no_port", "example.com", true},
		{"minimum", "example.com:1", true},
		{"http", "example.com:80", true},
		{"https", "example.com:443", true},
		{"maximum", "example.com:65535", true},
		{"leading_zeroes", "example.com:00080", true},
		{"ipv6", "[::1]", true},
		{"ipv6_port", "[2001:db8::1]:443", true},
		{"over_maximum", "example.com:65536", false},
		{"five_digits", "example.com:99999", false},
		{"overflow", "example.com:" + strings.Repeat("9", 100), false},
		{"zero", "example.com:0", false},
		{"empty", "example.com:", false},
		{"ipv6_zero", "[::1]:0", false},
		{"ipv6_empty", "[::1]:", false},
		{"ipv6_over_maximum", "[::1]:65536", false},
		{"negative", "example.com:-1", false},
		{"non_numeric", "example.com:abc", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			original := "https://" + tc.host + "/path"
			_, constructorErr := services.NewLinkService(nil, nil, original)
			if (constructorErr == nil) != tc.valid || (!tc.valid && !errors.Is(constructorErr, cerrors.ErrInvalidURL)) {
				t.Errorf("BASE_URL error = %v, valid = %v", constructorErr, tc.valid)
			}
			ctrl := gomock.NewController(t)
			repo := mocks.NewMockLinkRepository(ctrl)
			generator := mocks.NewMockCodeGenerator(ctrl)
			if tc.valid {
				generator.EXPECT().Generate().Return("abcdefghijkl", nil)
				repo.EXPECT().Save(gomock.Any(), models.Link{Code: "abcdefghijkl", URL: original}).Return(nil)
			}
			service, err := services.NewLinkService(repo, generator, "https://short.example")
			if err != nil {
				t.Fatal(err)
			}
			result, err := service.Shorten(context.Background(), original)
			if tc.valid {
				if err != nil || result != "https://short.example/abcdefghijkl" {
					t.Fatalf("Shorten = %q, %v", result, err)
				}
			} else if !errors.Is(err, cerrors.ErrInvalidURL) || result != "" {
				t.Fatalf("Shorten = %q, %v, want ErrInvalidURL", result, err)
			}
		})
	}
}

func TestLinkServiceResolve(t *testing.T) {
	failure := errors.New("database unavailable")
	for _, tt := range []struct {
		name, code       string
		canceled, call   bool
		repoErr, wantErr error
	}{
		{name: "success", code: "aB09_-cDeF12", call: true},
		{name: "missing", code: "abcdefghijkl", call: true, repoErr: cerrors.ErrLinkNotFound, wantErr: cerrors.ErrLinkNotFound},
		{name: "database failure", code: "abcdefghijkl", call: true, repoErr: failure, wantErr: failure},
		{name: "deadline", code: "abcdefghijkl", call: true, repoErr: context.DeadlineExceeded, wantErr: context.DeadlineExceeded},
		{name: "canceled", code: "abcdefghijkl", canceled: true, wantErr: context.Canceled},
		{name: "empty", wantErr: cerrors.ErrInvalidCode},
		{name: "short", code: "abc", wantErr: cerrors.ErrInvalidCode},
		{name: "long", code: "abcdefghijklm", wantErr: cerrors.ErrInvalidCode},
		{name: "slash", code: "abcdefghijk/", wantErr: cerrors.ErrInvalidCode},
		{name: "space", code: "abcdefghijk ", wantErr: cerrors.ErrInvalidCode},
		{name: "unicode", code: "abcdefghijя", wantErr: cerrors.ErrInvalidCode},
		{name: "trailing newline", code: "abcdefghijkl\n", wantErr: cerrors.ErrInvalidCode},
		{name: "embedded newline", code: "abcde\nghijkl", wantErr: cerrors.ErrInvalidCode},
		{name: "padding", code: "abcdefghijk=", wantErr: cerrors.ErrInvalidCode},
		{name: "base64 plus", code: "abcdefghijk+", wantErr: cerrors.ErrInvalidCode},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			repo := mocks.NewMockLinkRepository(ctrl)
			if tt.call {
				repo.EXPECT().FindByCode(gomock.Any(), tt.code).Return(models.Link{Code: tt.code, URL: "https://example.com/path?q=1#part"}, tt.repoErr)
			}
			service, err := services.NewLinkService(repo, mocks.NewMockCodeGenerator(ctrl), "https://short.example")
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if tt.canceled {
				cancel()
			}
			got, err := service.Resolve(ctx, tt.code)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
			want := ""
			if tt.wantErr == nil {
				want = "https://example.com/path?q=1#part"
			}
			if got != want {
				t.Fatalf("URL = %q, want %q", got, want)
			}
		})
	}
}

func TestRandomGenerate(t *testing.T) {
	code, err := (services.Random{}).Generate()
	if err != nil {
		t.Fatal(err)
	}
	if len(code) != models.CodeLength {
		t.Fatalf("code length = %d, want %d", len(code), models.CodeLength)
	}
	if _, err := base64.RawURLEncoding.DecodeString(code); err != nil {
		t.Fatalf("invalid base64url code: %v", err)
	}
}
