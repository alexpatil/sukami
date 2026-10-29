package gateway

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sukami/pkg/auth"
	"sukami/pkg/cache"
	"sukami/pkg/metrics"
	"sukami/pkg/proxy"
	"testing"
	"time"
)

func setupTestGateway() (*Gateway, auth.KeyStore, *metrics.Tracker) {
	c := cache.NewMemoryCache(time.Hour)
	tr := metrics.NewTracker(10)
	ks := auth.NewMemoryKeyStore()
	lim := auth.NewRateLimiter()
	h := proxy.NewHandler(c, tr, ks, lim, "https://api.openai.com/v1")
	gw := New(h, tr, ks, "master-secret")
	return gw, ks, tr
}

func TestGateway_HealthAndRoot(t *testing.T) {
	gw, _, _ := setupTestGateway()
	handler := gw.Router()

	// GET /health
	{
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK for /health, got %d", rec.Code)
		}
	}

	// GET /
	{
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK for /, got %d", rec.Code)
		}
		if !strings.Contains(rec.Body.String(), `"service":"sukami"`) {
			t.Errorf("expected service descriptor in body")
		}
	}
}

func TestGateway_MetricsEndpoints(t *testing.T) {
	gw, _, _ := setupTestGateway()
	handler := gw.Router()

	// GET /metrics (Prometheus)
	{
		req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK for /metrics, got %d", rec.Code)
		}
		if !strings.Contains(rec.Body.String(), "sukami_requests_total") {
			t.Errorf("expected prometheus exposition text")
		}
	}

	// GET /api/metrics (JSON summary)
	{
		req := httptest.NewRequest(http.MethodGet, "/api/metrics", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK for /api/metrics, got %d", rec.Code)
		}
	}
}

func TestGateway_KeysLifecycle(t *testing.T) {
	gw, _, _ := setupTestGateway()
	handler := gw.Router()

	// 1. Create key without master auth -> 401
	{
		req := httptest.NewRequest(http.MethodPost, "/api/keys", bytes.NewBufferString(`{"name":"Dev","tier":"standard"}`))
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 without master auth, got %d", rec.Code)
		}
	}

	// 2. Create key with master auth -> 201
	var createdKey auth.APIKey
	{
		req := httptest.NewRequest(http.MethodPost, "/api/keys", bytes.NewBufferString(`{"name":"Dev","tier":"standard"}`))
		req.Header.Set("Authorization", "Bearer master-secret")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created with master auth, got %d", rec.Code)
		}
		_ = json.Unmarshal(rec.Body.Bytes(), &createdKey)
	}

	// 3. List keys -> 200
	{
		req := httptest.NewRequest(http.MethodGet, "/api/keys", nil)
		req.Header.Set("Authorization", "Bearer master-secret")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK listing keys, got %d", rec.Code)
		}
	}

	// 4. Delete key -> 200
	{
		req := httptest.NewRequest(http.MethodDelete, "/api/keys?key="+createdKey.Key, nil)
		req.Header.Set("Authorization", "Bearer master-secret")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK deleting key, got %d", rec.Code)
		}
	}
}

func TestGateway_CorsHeaders(t *testing.T) {
	gw, _, _ := setupTestGateway()
	handler := gw.Router()

	req := httptest.NewRequest(http.MethodOptions, "/v1/chat/completions", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204 No Content for OPTIONS, got %d", rec.Code)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("expected Access-Control-Allow-Origin: *")
	}
}
