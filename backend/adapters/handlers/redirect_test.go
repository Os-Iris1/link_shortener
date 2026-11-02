package handlers_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"shorter/adapters/handlers"
	cerrors "shorter/core/errors"
	"shorter/core/ports/mocks"
)

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
		{name: "root", path: "/", status: 404},
		{name: "extra segment", path: "/abcdefghijkl/extra", status: 404},
		{name: "trailing slash", path: "/abcdefghijkl/", status: 404},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := mocks.NewMockLinkResolver(gomock.NewController(t))
			if tt.call {
				service.EXPECT().Resolve(gomock.Any(), tt.code).DoAndReturn(func(ctx context.Context, code string) (string, error) {
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
			mux := http.NewServeMux()
			mux.HandleFunc("GET /{code}", handlers.Redirect(service))
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
