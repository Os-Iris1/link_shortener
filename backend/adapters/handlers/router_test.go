package handlers_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.uber.org/mock/gomock"
	"shorter/adapters/handlers"
	"shorter/core/ports/mocks"
)

func TestRouterAPIMethods(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodHead, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions, http.MethodTrace} {
		t.Run(method, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			server := httptest.NewServer(handlers.NewRouter(mocks.NewMockLinkShortener(ctrl), mocks.NewMockLinkResolver(ctrl)))
			defer server.Close()
			request, err := http.NewRequest(method, server.URL+"/api/links?source=test", strings.NewReader(`{"url":"https://example.com"}`))
			if err != nil {
				t.Fatal(err)
			}
			request.Header.Set("Content-Type", "application/json")
			response, err := server.Client().Do(request)
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := response.Body.Close(); err != nil {
					t.Errorf("close response body: %v", err)
				}
			}()
			body, err := io.ReadAll(response.Body)
			if err != nil {
				t.Fatal(err)
			}
			if response.StatusCode != http.StatusMethodNotAllowed {
				t.Fatalf("status=%d, want 405", response.StatusCode)
			}
			if response.Header.Get("Allow") != "POST" {
				t.Fatalf("Allow=%q", response.Header.Get("Allow"))
			}
			if response.Header.Get("Content-Type") != "application/json; charset=utf-8" {
				t.Errorf("Content-Type=%q, want JSON", response.Header.Get("Content-Type"))
			}
			if method == http.MethodHead {
				if len(body) != 0 {
					t.Fatalf("HEAD body=%q", body)
				}
				return
			}
			var payload map[string]string
			if err := json.Unmarshal(body, &payload); err != nil {
				t.Fatalf("invalid JSON: %q: %v", body, err)
			}
			if payload["error"] != "method not allowed" {
				t.Fatalf("error=%q", payload["error"])
			}
		})
	}
}

func TestRouterHealth(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodHead} {
		t.Run(method, func(t *testing.T) {
			server := httptest.NewServer(handlers.NewRouter(nil, nil))
			defer server.Close()
			request, err := http.NewRequest(method, server.URL+"/healthz", nil)
			if err != nil {
				t.Fatal(err)
			}
			response, err := server.Client().Do(request)
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := response.Body.Close(); err != nil {
					t.Errorf("close response body: %v", err)
				}
			}()
			body, err := io.ReadAll(response.Body)
			if err != nil {
				t.Fatal(err)
			}
			want := "ok\n"
			if method == http.MethodHead {
				want = ""
			}
			if response.StatusCode != 200 || string(body) != want || response.Header.Get("Content-Type") != "text/plain; charset=utf-8" {
				t.Fatalf("health status=%d, body=%q", response.StatusCode, body)
			}
		})
	}
}
