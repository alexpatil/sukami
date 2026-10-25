package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sukami/pkg/auth"
	"sukami/pkg/cache"
	"sukami/pkg/config"
	"sukami/pkg/metrics"
	"sukami/pkg/proxy"
	"syscall"
	"time"
)

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func main() {
	// 0. Load local environment configuration if present
	_ = config.LoadEnv(".env")

	// 1. Initialize structured JSON logging
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	upstreamURL := os.Getenv("UPSTREAM_URL")
	if upstreamURL == "" {
		if os.Getenv("GROQ_API_KEY") != "" {
			upstreamURL = "https://api.groq.com/openai"
		} else {
			upstreamURL = "https://api.openai.com"
		}
	}

	// 2. Initialize Core Subsystems
	memoryCache := cache.NewMemoryCache(1 * time.Hour)
	tracker := metrics.NewTracker(50)
	authStore := auth.NewMemoryKeyStore()
	limiter := auth.NewRateLimiter()
	proxyHandler := proxy.NewHandler(memoryCache, tracker, authStore, limiter, upstreamURL)

	// Pre-seed a few simulated requests so the dashboard has rich immediate visuals
	seedDemoMetrics(tracker, memoryCache)

	// 3. Register modern Go 1.22+ ServeMux routes
	mux := http.NewServeMux()

	// OpenAI-compatible endpoints
	mux.HandleFunc("POST /v1/chat/completions", proxyHandler.HandleChatCompletions)
	mux.HandleFunc("GET /v1/models", proxyHandler.HandleModels)

	// SaaS Operations & Playground API
	mux.HandleFunc("GET /api/metrics", proxyHandler.HandleMetrics)
	mux.HandleFunc("POST /api/playground", proxyHandler.HandlePlayground)
	mux.HandleFunc("GET /api/keys", proxyHandler.HandleListKeys)
	mux.HandleFunc("POST /api/keys", proxyHandler.HandleCreateKey)
	mux.HandleFunc("DELETE /api/keys", proxyHandler.HandleDeleteKey)
	mux.HandleFunc("GET /api/config", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"supabase_url":      os.Getenv("SUPABASE_URL"),
			"supabase_anon_key": os.Getenv("SUPABASE_PUBLISHABLE_KEY"),
		})
	})
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ok","timestamp":"%s"}`, time.Now().UTC().Format(time.RFC3339))
	})

	// Developer Status Dashboard static files from ./web
	fs := http.FileServer(http.Dir("./web"))
	mux.Handle("GET /", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/v1/") || strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}
		fs.ServeHTTP(w, r)
	}))

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      corsMiddleware(mux),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// 4. Graceful Shutdown Listener
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		slog.Info("🚀 Sukami AI Token Optimizer Proxy started",
			"port", port,
			"upstream", upstreamURL,
			"dashboard", "http://localhost:"+port,
		)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("Server failed to start", "err", err)
			os.Exit(1)
		}
	}()

	<-stopChan
	slog.Info("Shutting down Sukami server gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("Server shutdown error", "err", err)
	} else {
		slog.Info("Server stopped cleanly.")
	}
}

func seedDemoMetrics(t *metrics.Tracker, c *cache.MemoryCache) {
	now := time.Now()
	t.RecordRequest(metrics.RequestRecord{
		ID:                "req_demo_1",
		Timestamp:         now.Add(-14 * time.Minute),
		Model:             "gpt-4o",
		PromptPreview:     "Could you please explain how to implement OAuth2 in Go...",
		OriginalTokens:    1420,
		SavedTokens:       520,
		InputSavedTokens:  180,
		OutputSavedTokens: 340,
		BilledTokens:      900,
		SavingsPercent:    36.6,
		CacheHit:          false,
		LatencyMs:         340,
		Status:            "200 OK (Optimized)",
	})
	t.RecordRequest(metrics.RequestRecord{
		ID:                "req_demo_2",
		Timestamp:         now.Add(-8 * time.Minute),
		Model:             "gpt-4o",
		PromptPreview:     "Could you please explain how to implement OAuth2 in Go...",
		OriginalTokens:    1420,
		SavedTokens:       1420,
		InputSavedTokens:  710,
		OutputSavedTokens: 710,
		BilledTokens:      0,
		SavingsPercent:    100.0,
		CacheHit:          true,
		LatencyMs:         8,
		Status:            "200 OK (Cache HIT)",
	})
	t.RecordRequest(metrics.RequestRecord{
		ID:                "req_demo_3",
		Timestamp:         now.Add(-2 * time.Minute),
		Model:             "claude-3-5-sonnet",
		PromptPreview:     "I would like you to analyze this PostgreSQL indexing query...",
		OriginalTokens:    3840,
		SavedTokens:       1150,
		InputSavedTokens:  420,
		OutputSavedTokens: 730,
		BilledTokens:      2690,
		SavingsPercent:    29.9,
		CacheHit:          false,
		LatencyMs:         420,
		Status:            "200 OK (Optimized)",
	})
}
