package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"shorter/adapters/handlers/dto"
	cerrors "shorter/core/errors"
	"shorter/core/ports"
)

func Redirect(service ports.LinkResolver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		originalURL, err := service.Resolve(ctx, r.PathValue("code"))
		switch {
		case errors.Is(err, cerrors.ErrInvalidCode):
			writeJSON(w, http.StatusBadRequest, dto.ErrorResponse{Error: "code must contain 12 base64url characters"})
		case errors.Is(err, cerrors.ErrLinkNotFound):
			writeJSON(w, http.StatusNotFound, dto.ErrorResponse{Error: "link not found"})
		case err != nil:
			slog.Error("Не удалось получить ссылку", "error", err)
			writeJSON(w, http.StatusInternalServerError, dto.ErrorResponse{Error: "internal server error"})
		default:
			http.Redirect(w, r, originalURL, http.StatusFound)
		}
	}
}
