package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.uber.org/mock/gomock"
	"shorter/adapters/handlers"
	"shorter/core/models"
	"shorter/core/ports/mocks"
	"shorter/core/services"
)

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
