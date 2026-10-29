package proxy

import (
	"fmt"
	"net/http"
	"time"
)

// CorsMiddleware wraps a handler with standard permissive CORS headers for client interoperability.
func CorsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, X-Upstream-Key, X-Sukami-Key, X-Upstream-Url, X-Base-Url")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// NewRouter builds and returns the standard HTTP request router for the Sukami gateway.
func NewRouter(h *Handler) http.Handler {
	mux := http.NewServeMux()

	// OpenAI-compatible endpoints
	mux.HandleFunc("POST /v1/chat/completions", h.HandleChatCompletions)
	mux.HandleFunc("GET /v1/models", h.HandleModels)

	// Observability & Multi-Tenant Keys
	mux.HandleFunc("GET /metrics", h.HandlePrometheusMetrics)
	mux.HandleFunc("GET /api/metrics", h.HandleMetrics)
	mux.HandleFunc("GET /api/keys", h.HandleListKeys)
	mux.HandleFunc("POST /api/keys", h.HandleCreateKey)
	mux.HandleFunc("DELETE /api/keys", h.HandleDeleteKey)
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ok","timestamp":"%s"}`+"\n", time.Now().UTC().Format(time.RFC3339))
	})
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"service":"sukami","version":"v0.1.0","status":"healthy","docs":"https://github.com/alexpatil/sukami"}`+"\n")
	})

	return CorsMiddleware(mux)
}
