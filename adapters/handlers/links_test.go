package handlers_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"shorter/adapters/handlers"
	cerrors "shorter/core/errors"
	"shorter/core/models"
	"shorter/core/ports/mocks"
	"shorter/core/services"
)

func TestRouterServesEmbeddedPage(t *testing.T) {
	ctrl := gomock.NewController(t)
	router := handlers.NewRouter(mocks.NewMockLinkShortener(ctrl), mocks.NewMockLinkResolver(ctrl))
	for _, tt := range []struct {
		path, contentType, content string
	}{
		{"/", "text/html", `/assets/app.js`},
		{"/assets/app.js", "text/javascript", `/api/links`},
		{"/assets/style.css", "text/css", `.shortener`},
	} {
		t.Run(tt.path, func(t *testing.T) {
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, tt.path, nil))
			if response.Code != http.StatusOK || !strings.HasPrefix(response.Header().Get("Content-Type"), tt.contentType) || !strings.Contains(response.Body.String(), tt.content) {
				t.Fatalf("GET %s: status=%d Content-Type=%q", tt.path, response.Code, response.Header().Get("Content-Type"))
			}
		})
	}
}

func TestShorten(t *testing.T) {
	for _, tt := range []struct {
		name, method, contentType, body string
		call                            bool
		url                             string
		serviceErr                      error
		status                          int
	}{
		{name: "created", body: `{"url":"https://example.com"}`, call: true, url: "https://example.com", status: 201},
		{name: "charset", contentType: "application/json; charset=utf-8", body: `{"url":"https://example.com"}`, call: true, url: "https://example.com", status: 201},
		{name: "invalid URL", body: `{"url":"bad"}`, call: true, url: "bad", serviceErr: cerrors.ErrInvalidURL, status: 400},
		{name: "missing URL", body: `{}`, call: true, serviceErr: cerrors.ErrInvalidURL, status: 400},
		{name: "database failure", body: `{"url":"https://example.com"}`, call: true, url: "https://example.com", serviceErr: errors.New("secret database details"), status: 500},
		{name: "broken JSON", body: `{`, status: 400},
		{name: "unknown field", body: `{"url":"https://example.com","other":1}`, status: 400},
		{name: "multiple objects", body: `{"url":"https://example.com"} {}`, status: 400},
		{name: "trailing garbage", body: `{"url":"https://example.com"} broken`, status: 400},
		{name: "wrong type", body: `{"url":42}`, status: 400},
		{name: "empty body", status: 400},
		{name: "too large", body: `{"url":"` + strings.Repeat("a", 65536) + `"}`, status: 413},
		{name: "wrong content type", contentType: "text/plain", body: `{}`, status: 415},
		{name: "method", method: http.MethodGet, status: 405},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := mocks.NewMockLinkShortener(gomock.NewController(t))
			if tt.call {
				service.EXPECT().Shorten(gomock.Any(), tt.url).Return("https://short.example/abc", tt.serviceErr)
			}
			method := tt.method
			if method == "" {
				method = http.MethodPost
			}
			contentType := tt.contentType
			if contentType == "" {
				contentType = "application/json"
			}
			req := httptest.NewRequest(method, "/api/links", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", contentType)
			response := httptest.NewRecorder()
			mux := handlers.NewRouter(service, nil)
			mux.ServeHTTP(response, req)
			if response.Code != tt.status {
				t.Fatalf("status = %d, want %d: %s", response.Code, tt.status, response.Body.String())
			}
			var body map[string]string
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if tt.status == 201 && body["short_url"] != "https://short.example/abc" {
				t.Fatalf("response = %v", body)
			}
			if tt.status >= 400 && body["error"] == "" {
				t.Fatalf("error missing: %v", body)
			}
			if strings.Contains(response.Body.String(), "secret database details") {
				t.Fatal("internal error exposed")
			}
		})
	}
}

func TestShortenRealServicePorts(t *testing.T) {
	for _, tc := range []struct {
		port   string
		status int
	}{
		{"65536", 400}, {"99999", 400}, {strings.Repeat("9", 100), 400}, {"0", 400}, {"", 400}, {"65535", 201},
	} {
		t.Run(tc.port, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := mocks.NewMockLinkRepository(ctrl)
			generator := mocks.NewMockCodeGenerator(ctrl)
			original := "https://example.com:" + tc.port + "/path?q=1#part"
			if tc.status == 201 {
				generator.EXPECT().Generate().Return("abcdefghijkl", nil)
				repo.EXPECT().Save(gomock.Any(), models.Link{Code: "abcdefghijkl", URL: original}).Return(nil)
			}
			service, err := services.NewLinkService(repo, generator, "https://short.example")
			if err != nil {
				t.Fatal(err)
			}
			request := httptest.NewRequest(http.MethodPost, "/api/links", strings.NewReader(`{"url":"`+original+`"}`))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			handlers.NewLinkHandler(service).Shorten(response, request)
			if response.Code != tc.status {
				t.Fatalf("status = %d, want %d", response.Code, tc.status)
			}
			var body map[string]string
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if tc.status == 400 && body["error"] == "" {
				t.Fatal("missing JSON error")
			}
			if tc.status == 201 && body["short_url"] != "https://short.example/abcdefghijkl" {
				t.Fatalf("unexpected body: %v", body)
			}
		})
	}
}

func TestRedirect(t *testing.T) {
	const target = "https://example.com/path?q=1&x=2#part"
	for _, tt := range []struct {
		name, method, path, code string
		call                     bool
		serviceErr               error
		status                   int
	}{
		{name: "found", path: "/aB09_-cDeF12", code: "aB09_-cDeF12", call: true, status: 302},
		{name: "query ignored", path: "/abcdefghijkl?url=https://other.example", code: "abcdefghijkl", call: true, status: 302},
		{name: "HEAD", method: "HEAD", path: "/abcdefghijkl", code: "abcdefghijkl", call: true, status: 302},
		{name: "invalid", path: "/bad", code: "bad", call: true, serviceErr: fmt.Errorf("validate: %w", cerrors.ErrInvalidCode), status: 400},
		{name: "missing", path: "/abcdefghijkl", code: "abcdefghijkl", call: true, serviceErr: fmt.Errorf("find: %w", cerrors.ErrLinkNotFound), status: 404},
		{name: "failure", path: "/abcdefghijkl", code: "abcdefghijkl", call: true, serviceErr: errors.New("secret database details"), status: 500},
		{name: "deadline", path: "/abcdefghijkl", code: "abcdefghijkl", call: true, serviceErr: context.DeadlineExceeded, status: 500},
		{name: "canceled", path: "/abcdefghijkl", code: "abcdefghijkl", call: true, serviceErr: context.Canceled, status: 500},
		{name: "POST", method: "POST", path: "/abcdefghijkl", status: 405},
		{name: "DELETE", method: "DELETE", path: "/abcdefghijkl", status: 405},
		{name: "extra segment", path: "/abcdefghijkl/extra", status: 404},
		{name: "trailing slash", path: "/abcdefghijkl/", status: 404},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := mocks.NewMockLinkResolver(gomock.NewController(t))
			if tt.call {
				service.EXPECT().Resolve(gomock.Any(), tt.code).DoAndReturn(func(ctx context.Context, _ string) (string, error) {
					deadline, ok := ctx.Deadline()
					if !ok || time.Until(deadline) > 5*time.Second {
						t.Fatal("missing operation timeout")
					}
					return target, tt.serviceErr
				})
			}
			method := tt.method
			if method == "" {
				method = "GET"
			}
			mux := handlers.NewRouter(nil, service)
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, httptest.NewRequest(method, tt.path, nil))
			if response.Code != tt.status {
				t.Fatalf("status = %d, want %d: %s", response.Code, tt.status, response.Body.String())
			}
			if tt.status == 302 {
				if response.Header().Get("Location") != target {
					t.Fatalf("Location = %q", response.Header().Get("Location"))
				}
				if method == "HEAD" && response.Body.Len() != 0 {
					t.Fatal("HEAD response has body")
				}
			} else if response.Header().Get("Location") != "" {
				t.Fatal("unexpected redirect on error")
			}
			if tt.status == 405 && response.Header().Get("Allow") != "GET, HEAD" {
				t.Fatalf("Allow = %q", response.Header().Get("Allow"))
			}
			if tt.call && tt.status >= 400 {
				var body map[string]string
				if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
					t.Fatal(err)
				}
				if body["error"] == "" || !strings.HasPrefix(response.Header().Get("Content-Type"), "application/json") {
					t.Fatal("missing JSON error")
				}
			}
			if strings.Contains(response.Body.String(), "secret database details") {
				t.Fatal("internal error exposed")
			}
		})
	}
}

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
