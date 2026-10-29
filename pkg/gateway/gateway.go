package gateway

import (
	"fmt"
	"net/http"
	"strings"
	"sukami/pkg/auth"
	"sukami/pkg/metrics"
	"sukami/pkg/proxy"
	"time"
)

// Gateway represents the HTTP entrypoint and administration server.
type Gateway struct {
	proxyHandler *proxy.Handler
	tracker      *metrics.Tracker
	authStore    auth.KeyStore
	masterKey    string
}

// New constructs an operational gateway router instance.
func New(proxyHandler *proxy.Handler, tracker *metrics.Tracker, authStore auth.KeyStore, masterKey string) *Gateway {
	return &Gateway{
		proxyHandler: proxyHandler,
		tracker:      tracker,
		authStore:    authStore,
		masterKey:    strings.TrimSpace(masterKey),
	}
}

// CorsMiddleware wraps HTTP handlers with standard cross-origin resource sharing headers.
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

// Router builds and returns the complete HTTP request multiplexer for the gateway.
func (gw *Gateway) Router() http.Handler {
	mux := http.NewServeMux()

	// OpenAI-compatible Proxy Data Plane
	mux.HandleFunc("POST /v1/chat/completions", gw.proxyHandler.HandleChatCompletions)
	mux.HandleFunc("GET /v1/models", gw.proxyHandler.HandleModels)

	// Control Plane & Observability
	mux.HandleFunc("GET /metrics", gw.HandlePrometheusMetrics)
	mux.HandleFunc("GET /api/metrics", gw.HandleMetrics)
	mux.HandleFunc("GET /api/keys", gw.HandleListKeys)
	mux.HandleFunc("POST /api/keys", gw.HandleCreateKey)
	mux.HandleFunc("DELETE /api/keys", gw.HandleDeleteKey)

	// System Health & Service Descriptor
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
