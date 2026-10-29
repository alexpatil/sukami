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
	"sukami/pkg/gateway"
	"sukami/pkg/metrics"
	"sukami/pkg/proxy"
	"syscall"
	"time"
)

func main() {
	// 0. Parse CLI Flags (Priority: CLI Flag > Environment Variable)
	portFlag := flag.String("port", "", "Port to run the gateway on (default: $PORT or 8080)")
	upstreamFlag := flag.String("upstream", "", "Upstream LLM API endpoint (e.g. $UPSTREAM_URL)")
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

	// Resolve Upstream URL (CLI Flag > Environment Variable)
	upstreamURL := *upstreamFlag
	if upstreamURL == "" {
		upstreamURL = os.Getenv("UPSTREAM_URL")
	}
	if upstreamURL == "" {
		slog.Warn("⚠️ No default UPSTREAM_URL configured. Incoming requests must provide X-Upstream-Url header.")
	}

	// 2. Initialize Core Subsystems
	memoryCache := cache.NewMemoryCache(1 * time.Hour)
	tracker := metrics.NewTracker(50)
	authStore := auth.NewMemoryKeyStore()
	limiter := auth.NewRateLimiter()
	proxyHandler := proxy.NewHandler(memoryCache, tracker, authStore, limiter, upstreamURL)
	masterKey := strings.TrimSpace(os.Getenv("SUKAMI_API_KEY"))
	if masterKey == "" {
		masterKey = strings.TrimSpace(os.Getenv("GATEWAY_KEY"))
	}
	if masterKey != "" {
		proxyHandler.SetGatewayKey(masterKey)
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

	// 3. Initialize Gateway Router
	gw := gateway.New(proxyHandler, tracker, authStore, masterKey)

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           gw.Router(),
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
