package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"time"

	"shorter/adapters/handlers/dto"
	cerrors "shorter/core/errors"
	"shorter/core/ports"
)

type LinkHandler struct {
	service ports.LinkShortener
}

func NewLinkHandler(service ports.LinkShortener) *LinkHandler {
	return &LinkHandler{service: service}
}

func (h *LinkHandler) Shorten(w http.ResponseWriter, r *http.Request) {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		writeJSON(w, http.StatusUnsupportedMediaType, dto.ErrorResponse{Error: "Content-Type must be application/json"})
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var request dto.ShortenRequest
	if err := decoder.Decode(&request); err != nil {
		writeDecodeError(w, err)
		return
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		writeDecodeError(w, err)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	shortURL, err := h.service.Shorten(ctx, request.URL)
	if errors.Is(err, cerrors.ErrInvalidURL) {
		writeJSON(w, http.StatusBadRequest, dto.ErrorResponse{Error: "url must be an absolute HTTP(S) URL without credentials, at most 8192 bytes"})
		return
	}
	if err != nil {
		slog.Error("Не удалось сократить ссылку", "error", err)
		writeJSON(w, http.StatusInternalServerError, dto.ErrorResponse{Error: "internal server error"})
		return
	}
	writeJSON(w, http.StatusCreated, dto.ShortenResponse{ShortURL: shortURL})
}

func writeDecodeError(w http.ResponseWriter, err error) {
	var sizeError *http.MaxBytesError
	if errors.As(err, &sizeError) {
		writeJSON(w, http.StatusRequestEntityTooLarge, dto.ErrorResponse{Error: "request body too large"})
		return
	}
	writeJSON(w, http.StatusBadRequest, dto.ErrorResponse{Error: "invalid JSON request"})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
