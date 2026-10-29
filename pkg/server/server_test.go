package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"sukami/pkg/auth"
	"sukami/pkg/cache"
	"sukami/pkg/metrics"
	"sukami/pkg/proxy"
	"sukami/pkg/server"
)

func setupTestServer(masterKey string) (*server.Server, *auth.MemoryKeyStore, *metrics.Tracker) {
	memCache := cache.NewMemoryCache(10 * time.Minute)
	tracker := metrics.NewTracker(10)
	keyStore := auth.NewMemoryKeyStore()
	limiter := auth.NewRateLimiter()
	proxyHandler := proxy.NewHandler(memCache, tracker, keyStore, limiter, "http://localhost:8080")

	srv := server.New(proxyHandler, tracker, keyStore, masterKey)
	return srv, keyStore, tracker
}

func TestServer_HealthAndRoot(t *testing.T) {
	srv, _, _ := setupTestServer("")
	router := srv.Router()

	// Test GET /health
	reqHealth := httptest.NewRequest(http.MethodGet, "/health", nil)
	recHealth := httptest.NewRecorder()
	router.ServeHTTP(recHealth, reqHealth)

	if recHealth.Code != http.StatusOK {
		t.Fatalf("expected /health status 200, got %d", recHealth.Code)
	}
	if !strings.Contains(recHealth.Body.String(), `"status":"ok"`) {
		t.Errorf("unexpected body: %s", recHealth.Body.String())
	}

	// Test GET /
	reqRoot := httptest.NewRequest(http.MethodGet, "/", nil)
	recRoot := httptest.NewRecorder()
	router.ServeHTTP(recRoot, reqRoot)

	if recRoot.Code != http.StatusOK {
		t.Fatalf("expected / status 200, got %d", recRoot.Code)
	}
	if !strings.Contains(recRoot.Body.String(), `"service":"sukami"`) {
		t.Errorf("unexpected body: %s", recRoot.Body.String())
	}
}

func TestServer_MetricsEndpoints(t *testing.T) {
	srv, _, tracker := setupTestServer("")
	tracker.RecordRequest(metrics.RequestRecord{
		OriginalTokens: 10,
		SavedTokens:    5,
		CostSavedUSD:   0.001,
		CacheHit:       false,
	})
	router := srv.Router()

	// Test /metrics (Prometheus format)
	reqProm := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	recProm := httptest.NewRecorder()
	router.ServeHTTP(recProm, reqProm)

	if recProm.Code != http.StatusOK {
		t.Fatalf("expected 200 for /metrics, got %d", recProm.Code)
	}
	if !strings.Contains(recProm.Body.String(), "sukami_requests_total") {
		t.Errorf("missing prometheus metric in output: %s", recProm.Body.String())
	}

	// Test /api/metrics (JSON format)
	reqJSON := httptest.NewRequest(http.MethodGet, "/api/metrics", nil)
	recJSON := httptest.NewRecorder()
	router.ServeHTTP(recJSON, reqJSON)

	if recJSON.Code != http.StatusOK {
		t.Fatalf("expected 200 for /api/metrics, got %d", recJSON.Code)
	}
	if !strings.Contains(recJSON.Body.String(), `"total_requests":1`) {
		t.Errorf("missing request count in metrics summary: %s", recJSON.Body.String())
	}
}

func TestServer_KeysLifecycle(t *testing.T) {
	masterKey := "super-secret-master-key"
	srv, _, _ := setupTestServer(masterKey)
	router := srv.Router()

	// 1. Unauthorized create
	createBody := `{"name":"test-tenant","tier":"pro"}`
	reqUnauth := httptest.NewRequest(http.MethodPost, "/api/keys", bytes.NewBufferString(createBody))
	recUnauth := httptest.NewRecorder()
	router.ServeHTTP(recUnauth, reqUnauth)

	if recUnauth.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without master key, got %d", recUnauth.Code)
	}

	// 2. Authorized create
	reqCreate := httptest.NewRequest(http.MethodPost, "/api/keys", bytes.NewBufferString(createBody))
	reqCreate.Header.Set("X-Sukami-Master-Key", masterKey)
	recCreate := httptest.NewRecorder()
	router.ServeHTTP(recCreate, reqCreate)

	if recCreate.Code != http.StatusCreated {
		t.Fatalf("expected 201 created, got %d: %s", recCreate.Code, recCreate.Body.String())
	}

	var createdKey struct {
		Key  string `json:"key"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal(recCreate.Body.Bytes(), &createdKey); err != nil {
		t.Fatalf("failed to parse json response: %v", err)
	}

	// 3. Authorized list
	reqList := httptest.NewRequest(http.MethodGet, "/api/keys", nil)
	reqList.Header.Set("X-Sukami-Master-Key", masterKey)
	recList := httptest.NewRecorder()
	router.ServeHTTP(recList, reqList)

	if recList.Code != http.StatusOK {
		t.Fatalf("expected 200 for list, got %d", recList.Code)
	}

	// 4. Authorized delete
	reqDel := httptest.NewRequest(http.MethodDelete, "/api/keys?key="+createdKey.Key, nil)
	reqDel.Header.Set("X-Sukami-Master-Key", masterKey)
	recDel := httptest.NewRecorder()
	router.ServeHTTP(recDel, reqDel)

	if recDel.Code != http.StatusOK {
		t.Fatalf("expected 200 for delete, got %d", recDel.Code)
	}
}

func TestServer_CorsHeaders(t *testing.T) {
	srv, _, _ := setupTestServer("")
	router := srv.Router()

	req := httptest.NewRequest(http.MethodOptions, "/v1/chat/completions", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204 for OPTIONS preflight, got %d", rec.Code)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("expected Access-Control-Allow-Origin: *")
	}
}
