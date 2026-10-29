package proxy

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sukami/pkg/auth"
	"sukami/pkg/metrics"
	"testing"
)

func TestHandleCreateKey_Standard(t *testing.T) {
	h, _, _ := setupTestHandler()

	body := []byte(`{"name":"Developer Key","tier":"standard"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/keys", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	h.HandleCreateKey(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201 Created, got %d", rec.Code)
	}

	var created auth.APIKey
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if created.Tier != "standard" {
		t.Errorf("expected tier 'standard', got %s", created.Tier)
	}
	if created.RateLimitPerMin != 60 {
		t.Errorf("expected RateLimitPerMin 60, got %d", created.RateLimitPerMin)
	}
}

func TestHandleMetrics_Summary(t *testing.T) {
	h, _, _ := setupTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/metrics", nil)
	rec := httptest.NewRecorder()

	h.HandleMetrics(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 OK, got %d", rec.Code)
	}

	var summary metrics.Summary
	if err := json.Unmarshal(rec.Body.Bytes(), &summary); err != nil {
		t.Fatalf("failed to decode metrics summary: %v", err)
	}
}

func TestHandlePrometheusMetrics(t *testing.T) {
	h, _, _ := setupTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()

	h.HandlePrometheusMetrics(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 OK, got %d", rec.Code)
	}

	contentType := rec.Header().Get("Content-Type")
	if !strings.HasPrefix(contentType, "text/plain") {
		t.Errorf("expected text/plain Content-Type, got %s", contentType)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "sukami_requests_total") {
		t.Errorf("expected body to contain 'sukami_requests_total', got:\n%s", body)
	}
	if !strings.Contains(body, "sukami_cost_saved_usd_total") {
		t.Errorf("expected body to contain 'sukami_cost_saved_usd_total', got:\n%s", body)
	}
}
