package handlers

import (
	"embed"
	"net/http"

	"shorter/adapters/handlers/dto"
	"shorter/core/ports"
)

//go:embed web/index.html web/app.js web/style.css
var webFiles embed.FS //nolint:gochecknoglobals // Embedded assets are read-only.

func NewRouter(shortener ports.LinkShortener, resolver ports.LinkResolver) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFileFS(w, r, webFiles, "web/index.html")
	})
	mux.HandleFunc("GET /assets/app.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		http.ServeFileFS(w, r, webFiles, "web/app.js")
	})
	mux.HandleFunc("GET /assets/style.css", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
		http.ServeFileFS(w, r, webFiles, "web/style.css")
	})
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
