package test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sukami/pkg/auth"
	"sukami/pkg/cache"
	"sukami/pkg/metrics"
	"sukami/pkg/proxy"
	"sync/atomic"
	"testing"
	"time"
)

func setupE2EEnvironment(t *testing.T) (upstreamServer *httptest.Server, gatewayServer *httptest.Server, upstreamCallCounter *int64) {
	var callCount int64

	// 1. Mock Upstream LLM Server
	upstreamServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":{"message":"Missing authorization"}}`))
			return
		}

		if r.URL.Path == "/v1/models" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"object":"list","data":[{"id":"mock-gpt-4o","object":"model"}]}`))
			return
		}

		if r.URL.Path == "/v1/chat/completions" {
			atomic.AddInt64(&callCount, 1)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": "chatcmpl-e2e-123",
				"object": "chat.completion",
				"created": 1700000000,
				"model": "mock-gpt-4o",
				"choices": [
					{
						"index": 0,
						"message": {"role": "assistant", "content": "Concurrency is the composition of independently executing computations."},
						"finish_reason": "stop"
					}
				],
				"usage": {
					"prompt_tokens": 15,
					"completion_tokens": 10,
					"total_tokens": 25
				}
			}`))
			return
		}

		http.NotFound(w, r)
	}))

	// 2. Sukami Gateway Server
	memCache := cache.NewMemoryCache(1 * time.Hour)
	tracker := metrics.NewTracker(50)
	keyStore := auth.NewMemoryKeyStore()
	limiter := auth.NewRateLimiter()
	proxyHandler := proxy.NewHandler(memCache, tracker, keyStore, limiter, upstreamServer.URL)
	gatewayRouter := proxy.NewRouter(proxyHandler)
	gatewayServer = httptest.NewServer(gatewayRouter)

	return upstreamServer, gatewayServer, &callCount
}

func TestE2E_FullGatewayLifecycle(t *testing.T) {
	upstreamServer, gatewayServer, callCounter := setupE2EEnvironment(t)
	defer upstreamServer.Close()
	defer gatewayServer.Close()

	client := gatewayServer.Client()

	// 1. Health and Root JSON Descriptors
	{
		resp, err := client.Get(gatewayServer.URL + "/health")
		if err != nil || resp.StatusCode != http.StatusOK {
			t.Fatalf("GET /health failed: status %d, err %v", resp.StatusCode, err)
		}
		resp.Body.Close()

		respRoot, err := client.Get(gatewayServer.URL + "/")
		if err != nil || respRoot.StatusCode != http.StatusOK {
			t.Fatalf("GET / failed: status %d, err %v", respRoot.StatusCode, err)
		}
		bodyBytes, _ := io.ReadAll(respRoot.Body)
		respRoot.Body.Close()
		if !strings.Contains(string(bodyBytes), `"service":"sukami"`) {
			t.Fatalf("expected service descriptor, got %s", string(bodyBytes))
		}
	}

	// 2. Transparent /v1/models Proxying
	{
		req, _ := http.NewRequest(http.MethodGet, gatewayServer.URL+"/v1/models", nil)
		req.Header.Set("Authorization", "Bearer test-upstream-key")
		resp, err := client.Do(req)
		if err != nil || resp.StatusCode != http.StatusOK {
			t.Fatalf("GET /v1/models failed: status %d, err %v", resp.StatusCode, err)
		}
		modelsBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if !strings.Contains(string(modelsBody), "mock-gpt-4o") {
			t.Fatalf("expected mock-gpt-4o model, got %s", string(modelsBody))
		}
	}

	// 3. Chat Completion: Request 1 (Cache MISS)
	chatPayload := `{"model":"mock-gpt-4o","messages":[{"role":"user","content":"Explain Go concurrency channels in simple words."}]}`
	{
		req, _ := http.NewRequest(http.MethodPost, gatewayServer.URL+"/v1/chat/completions", strings.NewReader(chatPayload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer test-upstream-key")
		resp, err := client.Do(req)
		if err != nil || resp.StatusCode != http.StatusOK {
			t.Fatalf("Chat Request 1 failed: status %d, err %v", resp.StatusCode, err)
		}

		cacheHeader := resp.Header.Get("X-Cache")
		resp.Body.Close()

		if cacheHeader != "MISS" {
			t.Fatalf("expected X-Cache MISS on first query, got %s", cacheHeader)
		}
		if atomic.LoadInt64(callCounter) != 1 {
			t.Fatalf("expected upstream call count 1, got %d", atomic.LoadInt64(callCounter))
		}
	}

	// 4. Chat Completion: Request 2 (Cache HIT - Zero Upstream Overhead)
	{
		req, _ := http.NewRequest(http.MethodPost, gatewayServer.URL+"/v1/chat/completions", strings.NewReader(chatPayload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer test-upstream-key")
		resp, err := client.Do(req)
		if err != nil || resp.StatusCode != http.StatusOK {
			t.Fatalf("Chat Request 2 failed: status %d, err %v", resp.StatusCode, err)
		}

		cacheHeader := resp.Header.Get("X-Cache")
		resp.Body.Close()

		if cacheHeader != "HIT" {
			t.Fatalf("expected X-Cache HIT on repeated query, got %s", cacheHeader)
		}
		// Upstream MUST NOT have been called again!
		if atomic.LoadInt64(callCounter) != 1 {
			t.Fatalf("upstream was called on cache hit! call count = %d", atomic.LoadInt64(callCounter))
		}
	}

	// 5. Prometheus Observability Endpoint
	{
		resp, err := client.Get(gatewayServer.URL + "/metrics")
		if err != nil || resp.StatusCode != http.StatusOK {
			t.Fatalf("GET /metrics failed: status %d, err %v", resp.StatusCode, err)
		}
		metricsText, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if !strings.Contains(string(metricsText), "sukami_requests_total") {
			t.Fatalf("expected prometheus metrics in response")
		}
		if !strings.Contains(string(metricsText), "sukami_cache_hits_total") {
			t.Fatalf("expected cache hits in prometheus metrics")
		}
	}

	// 6. Multi-Tenant Key Lifecycle & Rate Limiting
	{
		createKeyBody := `{"name":"E2E Tenant","tier":"free"}`
		respKey, err := client.Post(gatewayServer.URL+"/api/keys", "application/json", bytes.NewBufferString(createKeyBody))
		if err != nil || respKey.StatusCode != http.StatusCreated {
			t.Fatalf("POST /api/keys failed: %v", err)
		}
		var createdKey struct {
			Key string `json:"key"`
		}
		_ = json.NewDecoder(respKey.Body).Decode(&createdKey)
		respKey.Body.Close()

		if createdKey.Key == "" {
			t.Fatalf("expected non-empty tenant key created")
		}
	}
}
