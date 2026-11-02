package services_test

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/mock/gomock"

	cerrors "shorter/core/errors"
	"shorter/core/models"
	"shorter/core/ports/mocks"
	"shorter/core/services"
)

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
