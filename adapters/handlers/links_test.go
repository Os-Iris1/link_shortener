package handlers_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.uber.org/mock/gomock"

	"shorter/adapters/handlers"
	cerrors "shorter/core/errors"
	"shorter/core/ports/mocks"
)

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
			mux := http.NewServeMux()
			mux.HandleFunc("POST /api/links", handlers.NewLinkHandler(service).Shorten)
			mux.ServeHTTP(response, req)
			if response.Code != tt.status {
				t.Fatalf("status = %d, want %d: %s", response.Code, tt.status, response.Body.String())
			}
			if tt.status == 405 {
				return
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
