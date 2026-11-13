package main

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"sukami/pkg/auth"
	"sukami/pkg/cache"
	"sukami/pkg/config"
	"sukami/pkg/metrics"
	"sukami/pkg/proxy"
	"time"
)

const benchPrompt = `Provide a comprehensive, high-level comparative analysis between PostgreSQL and MongoDB for an enterprise FinTech banking platform.
Cover:
1. ACID transactions and multi-document consistency guarantees.
2. Horizontal sharding vs Read-replicas and connection pooling trade-offs under heavy financial transaction loads.
3. Financial compliance, immutable audit logging, and point-in-time recovery.
4. Definitive architectural verdict: Which database should be the primary system of record, and which for secondary analytics?
Provide an executive, thorough, and definitive report concluding with a clear summary verdict.`

type benchConfig struct {
	upstreamURL string
	apiKey      string
	model       string
	runs        int
}

func loadConfig() (benchConfig, error) {
	cfg := benchConfig{
		upstreamURL: os.Getenv("UPSTREAM_URL"),
		apiKey:      os.Getenv("UPSTREAM_API_KEY"),
		model:       os.Getenv("BENCH_MODEL"),
		runs:        3,
	}
	for name, value := range map[string]string{"UPSTREAM_URL": cfg.upstreamURL, "UPSTREAM_API_KEY": cfg.apiKey, "BENCH_MODEL": cfg.model} {
		if value == "" {
			return cfg, fmt.Errorf("%s is not set (in .env or the environment)", name)
		}
	}
	if raw := os.Getenv("BENCH_RUNS"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 {
			return cfg, fmt.Errorf("BENCH_RUNS must be a positive integer, got %q", raw)
		}
		cfg.runs = n
	}
	return cfg, nil
}

// startProxy serves the real chat completions handler on a loopback port with an empty cache.
func startProxy(upstreamURL string) (url string, stop func(), err error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", nil, err
	}
	handler := proxy.NewHandler(cache.NewMemoryCache(time.Hour), metrics.NewTracker(10), auth.NewMemoryKeyStore(), auth.NewRateLimiter(), upstreamURL)
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/chat/completions", handler.HandleChatCompletions)
	server := &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go func() { _ = server.Serve(listener) }()
	return "http://" + listener.Addr().String() + "/v1/chat/completions", func() { _ = server.Close() }, nil
}

// runOnce measures the four cases once. Each proxied case starts from an empty cache.
func runOnce(cfg benchConfig, client *http.Client) (map[string]sample, error) {
	out := map[string]sample{}
	record := func(name, url string, headers map[string]string, wantCache string) error {
		s, err := call(client, url, cfg.apiKey, cfg.model, headers)
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		if wantCache != "" && s.cache != wantCache {
			return fmt.Errorf("%s: expected X-Cache=%q, got %q", name, wantCache, s.cache)
		}
		out[name] = s
		return nil
	}

	if err := record(caseDirect, proxy.CompletionsURL(cfg.upstreamURL), nil, ""); err != nil {
		return nil, err
	}

	proxyURL, stop, err := startProxy(cfg.upstreamURL)
	if err != nil {
		return nil, err
	}
	defer stop()
	if err := record(caseProxy, proxyURL, nil, "MISS"); err != nil {
		return nil, err
	}
	if err := record(caseCacheHit, proxyURL, nil, "HIT"); err != nil {
		return nil, err
	}

	denseURL, stopDense, err := startProxy(cfg.upstreamURL)
	if err != nil {
		return nil, err
	}
	defer stopDense()
	return out, record(caseDensity, denseURL, map[string]string{"X-Sukami-Density": "true"}, "MISS")
}

func main() {
	_ = config.LoadEnv(".env")
	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}

	client := &http.Client{Timeout: 180 * time.Second}
	results := map[string][]sample{}
	for i := 1; i <= cfg.runs; i++ {
		fmt.Fprintf(os.Stderr, "run %d of %d\n", i, cfg.runs)
		run, err := runOnce(cfg, client)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}
		for name, s := range run {
			results[name] = append(results[name], s)
		}
	}
	fmt.Print(renderReport(cfg, results, time.Now()))
}
