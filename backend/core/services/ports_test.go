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
