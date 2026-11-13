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
	"sukami/pkg/api"
	"sukami/pkg/auth"
	"sukami/pkg/cache"
	"sukami/pkg/config"
	"sukami/pkg/metrics"
	"sukami/pkg/proxy"
	"syscall"
	"time"
)

func main() {
	// CLI flags take priority over environment variables.
	portFlag := flag.String("port", "", "Port to run the server on (default: $PORT or 8080)")
	upstreamFlag := flag.String("upstream", "", "Upstream LLM API endpoint (e.g. $UPSTREAM_URL)")
	versionFlag := flag.Bool("version", false, "Print binary version and exit")
	flag.StringVar(portFlag, "p", "", "Shorthand for --port")
	flag.BoolVar(versionFlag, "v", false, "Shorthand for --version")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Sukami AI Token Optimization Gateway\n\nUsage:\n  sukami [flags]\n\nFlags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *versionFlag {
		fmt.Printf("sukami version %s\n", api.Version)
		return
	}

	// Optional local environment file
	_ = config.LoadEnv(".env")

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
		slog.Info("🔒 Cloud Security Mode: Master authentication active")
	}

	// Model pricing is fetched from the pricing API at boot.
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if count, err := metrics.FetchLiveRates(ctx); err != nil {
			slog.Warn("Pricing API unavailable (running in offline mode)", "err", err)
		} else {
			slog.Info("✅ Synchronized live model rates dynamically from API", "models", count)
		}
	}()

	apiServer := api.New(proxyHandler, tracker, authStore, masterKey)

	httpServer := &http.Server{
		Addr:              ":" + port,
		Handler:           apiServer.Router(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      120 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	// Graceful shutdown listener
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		slog.Info("🚀 Sukami AI Token Optimizer Proxy started",
			"port", port,
			"upstream", upstreamURL,
		)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("Server failed to start", "err", err)
			os.Exit(1)
		}
	}()

	<-stopChan
	slog.Info("Shutting down Sukami server gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("Server shutdown error", "err", err)
	} else {
		slog.Info("Server stopped cleanly.")
	}
}
