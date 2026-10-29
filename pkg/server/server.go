package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"sukami/pkg/auth"
	"sukami/pkg/metrics"
	"sukami/pkg/proxy"
)

// Server represents the HTTP entrypoint and routing engine.
type Server struct {
	proxyHandler *proxy.Handler
	tracker      *metrics.Tracker
	authStore    auth.KeyStore
	masterKey    string
}

// New constructs an operational server instance.
func New(proxyHandler *proxy.Handler, tracker *metrics.Tracker, authStore auth.KeyStore, masterKey string) *Server {
	return &Server{
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

// Router builds and returns the complete HTTP request multiplexer for the server.
func (s *Server) Router() http.Handler {
	mux := http.NewServeMux()

	// OpenAI-compatible Proxy Data Plane
	mux.HandleFunc("POST /v1/chat/completions", s.proxyHandler.HandleChatCompletions)
	mux.HandleFunc("GET /v1/models", s.proxyHandler.HandleModels)

	// Observability & Metrics
	mux.HandleFunc("GET /metrics", s.HandlePrometheusMetrics)
	mux.HandleFunc("GET /api/metrics", s.HandleMetrics)
	mux.HandleFunc("GET /api/keys", s.HandleListKeys)
	mux.HandleFunc("POST /api/keys", s.HandleCreateKey)
	mux.HandleFunc("DELETE /api/keys", s.HandleDeleteKey)

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

// HandleMetrics returns operational metrics summary.
func (s *Server) HandleMetrics(w http.ResponseWriter, r *http.Request) {
	summary := s.tracker.GetSummary()
	writeJSON(w, http.StatusOK, summary)
}

// HandlePrometheusMetrics returns standard Prometheus exposition format text.
func (s *Server) HandlePrometheusMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(s.tracker.FormatPrometheus()))
}

func (s *Server) isMasterAuth(r *http.Request) bool {
	if s.masterKey == "" {
		return true
	}
	for _, header := range []string{"X-Sukami-Master-Key", "X-Sukami-Key"} {
		if val := strings.TrimSpace(r.Header.Get(header)); val != "" {
			if strings.TrimPrefix(val, "Bearer ") == s.masterKey {
				return true
			}
		}
	}
	rawAuth := strings.TrimSpace(r.Header.Get("Authorization"))
	return strings.TrimPrefix(rawAuth, "Bearer ") == s.masterKey
}

// CreateKeyRequest represents incoming payload for new tenant key.
type CreateKeyRequest struct {
	Name string `json:"name"`
	Tier string `json:"tier"`
}

// HandleListKeys returns all registered tenant API keys.
func (s *Server) HandleListKeys(w http.ResponseWriter, r *http.Request) {
	if !s.isMasterAuth(r) {
		writeError(w, http.StatusUnauthorized, "Unauthorized: Master API key required")
		return
	}
	keys := s.authStore.ListKeys()
	writeJSON(w, http.StatusOK, keys)
}

// HandleCreateKey provisions a new tenant API key.
func (s *Server) HandleCreateKey(w http.ResponseWriter, r *http.Request) {
	if !s.isMasterAuth(r) {
		writeError(w, http.StatusUnauthorized, "Unauthorized: Master API key required")
		return
	}
	var req CreateKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	key, err := s.authStore.CreateKey(req.Name, req.Tier)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, key)
}

// HandleDeleteKey revokes an API key.
func (s *Server) HandleDeleteKey(w http.ResponseWriter, r *http.Request) {
	if !s.isMasterAuth(r) {
		writeError(w, http.StatusUnauthorized, "Unauthorized: Master API key required")
		return
	}
	key := r.URL.Query().Get("key")
	if key == "" {
		writeError(w, http.StatusBadRequest, "Missing key query parameter")
		return
	}

	revoked := s.authStore.RevokeKey(key)
	writeJSON(w, http.StatusOK, map[string]any{"revoked": revoked})
}

// writeJSON writes a JSON response with status code.
func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// writeError writes a standard JSON error message.
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]string{
			"message": message,
		},
	})
}
