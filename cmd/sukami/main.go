package main

import (
	"context"
	"errors"
	"flag"
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
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, X-Upstream-Key, X-Sukami-Key, X-Provider, X-Upstream-Url, X-Base-Url")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func main() {
	// 0. Parse CLI Flags (Priority: CLI Flag > Environment Variable > Default)
	portFlag := flag.String("port", "", "Port to run the gateway on (default: $PORT or 8080)")
	upstreamFlag := flag.String("upstream", "", "Upstream LLM API endpoint (default: $UPSTREAM_URL or https://api.openai.com)")
	flag.StringVar(portFlag, "p", "", "Shorthand for --port")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Sukami AI Token Optimization Gateway\n\nUsage:\n  sukami [flags]\n\nFlags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	// Optional local environment file
	_ = config.LoadEnv(".env")

	// 1. Initialize structured logging
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Resolve Port (Flag > Env > Default 8080)
	port := *portFlag
	if port == "" {
		port = os.Getenv("PORT")
	}
	if port == "" {
		port = "8080"
	}

	// Resolve Upstream URL (CLI Flag > Environment Variable > Default)
	upstreamURL := *upstreamFlag
	if upstreamURL == "" {
		upstreamURL = os.Getenv("UPSTREAM_URL")
	}
	if upstreamURL == "" {
		upstreamURL = proxy.DefaultUpstreamURL
	}

	// 2. Initialize Core Subsystems
	memoryCache := cache.NewMemoryCache(1 * time.Hour)
	tracker := metrics.NewTracker(50)
	authStore := auth.NewMemoryKeyStore()
	limiter := auth.NewRateLimiter()
	proxyHandler := proxy.NewHandler(memoryCache, tracker, authStore, limiter, upstreamURL)
	if gwKey := strings.TrimSpace(os.Getenv("SUKAMI_API_KEY")); gwKey != "" {
		proxyHandler.SetGatewayKey(gwKey)
		slog.Info("🔒 Cloud Security Mode: Gateway master authentication active")
	} else if gwKey := strings.TrimSpace(os.Getenv("GATEWAY_KEY")); gwKey != "" {
		proxyHandler.SetGatewayKey(gwKey)
		slog.Info("🔒 Cloud Security Mode: Gateway master authentication active")
	}

	// Fetch live model pricing dynamically from pricing API on boot (zero hardcoding)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if count, err := metrics.FetchLiveRates(ctx); err != nil {
			slog.Warn("Pricing API unavailable (running in offline mode)", "err", err)
		} else {
			slog.Info("✅ Synchronized live model rates dynamically from API", "models", count)
		}
	}()

	// 3. Register modern Go 1.22+ ServeMux routes
	mux := http.NewServeMux()

	// OpenAI-compatible endpoints
	mux.HandleFunc("POST /v1/chat/completions", proxyHandler.HandleChatCompletions)
	mux.HandleFunc("GET /v1/models", proxyHandler.HandleModels)
	mux.HandleFunc("POST /api/models/fetch", proxyHandler.HandleFetchProviderModels)

	// SaaS Operations & Playground API
	mux.HandleFunc("GET /metrics", proxyHandler.HandlePrometheusMetrics)
	mux.HandleFunc("GET /api/metrics", proxyHandler.HandleMetrics)
	mux.HandleFunc("POST /api/playground", proxyHandler.HandlePlayground)
	mux.HandleFunc("GET /api/keys", proxyHandler.HandleListKeys)
	mux.HandleFunc("POST /api/keys", proxyHandler.HandleCreateKey)
	mux.HandleFunc("DELETE /api/keys", proxyHandler.HandleDeleteKey)
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
		Addr:              ":" + port,
		Handler:           corsMiddleware(mux),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      120 * time.Second,
		IdleTimeout:       120 * time.Second,
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
