package proxy

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestHandleModels_UnauthorizedWithoutGatewayAuth(t *testing.T) {
	h, _, _ := setupTestHandler()
	_ = os.Setenv("REQUIRE_AUTH", "true")
	defer os.Unsetenv("REQUIRE_AUTH")

	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	rec := httptest.NewRecorder()

	h.HandleModels(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", rec.Code)
	}
}

func TestHandleModels_MissingUpstreamAuth(t *testing.T) {
	h, _, _ := setupTestHandler()
	_ = os.Unsetenv("UPSTREAM_API_KEY")

	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	rec := httptest.NewRecorder()

	h.HandleModels(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for missing upstream auth, got %d", rec.Code)
	}
}

func TestHandleModels_TransparentProxy(t *testing.T) {
	mockUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Errorf("expected path /v1/models, got %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer secret-upstream-key" {
			t.Errorf("missing or invalid authorization header: %s", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Custom-Provider-Header", "active")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"object":"list","data":[{"id":"upstream-model-v1","object":"model"}]}`))
	}))
	defer mockUpstream.Close()

	h, _, _ := setupTestHandler()
	h.upstreamURL = mockUpstream.URL

	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	req.Header.Set("Authorization", "Bearer secret-upstream-key")
	rec := httptest.NewRecorder()

	h.HandleModels(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}
	if rec.Header().Get("X-Custom-Provider-Header") != "active" {
		t.Errorf("expected custom header to be preserved in proxy response")
	}

	var resp struct {
		Object string `json:"object"`
		Data   []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode models JSON: %v", err)
	}
	if len(resp.Data) != 1 || resp.Data[0].ID != "upstream-model-v1" {
		t.Errorf("expected upstream-model-v1 in data, got %+v", resp.Data)
	}
}

func TestHandleModels_WithCustomUpstreamHeader(t *testing.T) {
	customServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/models") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"object":"list","data":[{"id":"custom-node-model","object":"model"}]}`))
	}))
	defer customServer.Close()

	h, _, _ := setupTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	req.Header.Set("Authorization", "Bearer test-key")
	req.Header.Set("X-Upstream-Url", customServer.URL)
	rec := httptest.NewRecorder()

	h.HandleModels(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "custom-node-model") {
		t.Errorf("expected custom-node-model in response, got %s", rec.Body.String())
	}
}

