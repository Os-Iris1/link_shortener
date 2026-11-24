package handlers

import (
	"net/http"

	"shorter/adapters/handlers/dto"
	"shorter/core/ports"
)

func NewRouter(shortener ports.LinkShortener, resolver ports.LinkResolver) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/links", NewLinkHandler(shortener).Shorten)
	mux.HandleFunc("/api/links", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Allow", http.MethodPost)
		writeJSON(w, http.StatusMethodNotAllowed, dto.ErrorResponse{Error: "method not allowed"})
	})
	mux.HandleFunc("GET /{code}", Redirect(resolver))
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("ok\n"))
	})
	return mux
}
