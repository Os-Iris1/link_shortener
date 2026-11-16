package integration_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

// TestLongRedirects runs against an explicitly supplied disposable Compose stack.
// It creates links; never point SHORTER_TEST_PROXY_URL at a production database.
func TestLongRedirects(t *testing.T) {
	base := os.Getenv("SHORTER_TEST_PROXY_URL")
	if base == "" {
		t.Skip("set SHORTER_TEST_PROXY_URL to an isolated nginx endpoint")
	}
	client := &http.Client{Timeout: 10 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	prefix, suffix := "https://example.com/", "?q=я&encoded=%23#я"
	unicodeURL := prefix + strings.Repeat("я", (8192-len(prefix)-len(suffix))/2) + suffix
	unicodeURL += strings.Repeat("x", 8192-len(unicodeURL))
	for _, tc := range []struct {
		name     string
		original string
		status   int
	}{
		{"ascii_limit", prefix + strings.Repeat("a", 8192-len(prefix)), http.StatusCreated},
		{"unicode_limit", unicodeURL, http.StatusCreated},
		{"over_limit", prefix + strings.Repeat("a", 8193-len(prefix)), http.StatusBadRequest},
	} {
		t.Run(tc.name, func(t *testing.T) {
			payload, err := json.Marshal(map[string]string{"url": tc.original})
			if err != nil {
				t.Fatal(err)
			}
			response, err := client.Post(base+"/api/links", "application/json", bytes.NewReader(payload))
			if err != nil {
				t.Fatal(err)
			}
			defer response.Body.Close()
			if response.StatusCode != tc.status {
				t.Fatalf("POST status = %d, want %d", response.StatusCode, tc.status)
			}
			if tc.status != http.StatusCreated {
				return
			}
			var result struct {
				ShortURL string `json:"short_url"`
			}
			if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
				t.Fatal(err)
			}
			short, err := url.Parse(result.ShortURL)
			if err != nil {
				t.Fatal(err)
			}
			if len(strings.TrimPrefix(short.Path, "/")) != 12 {
				t.Fatal("invalid short code")
			}
			var expected strings.Builder
			for _, b := range []byte(tc.original) {
				if b >= 128 {
					fmt.Fprintf(&expected, "%%%02x", b)
				} else {
					expected.WriteByte(b)
				}
			}
			for _, method := range []string{http.MethodGet, http.MethodHead} {
				t.Run(method, func(t *testing.T) {
					request, err := http.NewRequest(method, base+short.EscapedPath(), nil)
					if err != nil {
						t.Fatal(err)
					}
					redirect, err := client.Do(request)
					if err != nil {
						t.Fatal(err)
					}
					defer redirect.Body.Close()
					if redirect.StatusCode != http.StatusFound {
						t.Fatalf("status = %d, want 302", redirect.StatusCode)
					}
					if redirect.Header.Get("Location") != expected.String() {
						t.Fatalf("Location differs: got %d bytes, want %d", len(redirect.Header.Get("Location")), expected.Len())
					}
					if method == http.MethodHead {
						body, err := io.ReadAll(redirect.Body)
						if err != nil || len(body) != 0 {
							t.Fatalf("HEAD body = %q, error = %v", body, err)
						}
					}
				})
			}
		})
	}
}
