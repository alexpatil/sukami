package proxy

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// HandleModels transparently forwards the /v1/models query to the upstream provider.
func (h *Handler) HandleModels(w http.ResponseWriter, r *http.Request) {
	authorized, _ := h.validateGatewayAuth(r)
	if !authorized {
		writeError(w, http.StatusUnauthorized, "Unauthorized: Invalid or missing gateway API key")
		return
	}

	authHeader := h.resolveUpstreamAuth(r)
	if authHeader == "" {
		writeError(w, http.StatusUnauthorized, "Missing upstream authorization: pass Authorization: Bearer <key> or set UPSTREAM_API_KEY")
		return
	}

	targetBase := h.upstreamURL
	if custom := r.Header.Get("X-Upstream-Url"); custom != "" {
		targetBase = custom
	} else if custom := r.Header.Get("X-Base-Url"); custom != "" {
		targetBase = custom
	} else if providerID := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Provider"))); providerID != "" {
		if p, ok := knownProviders[providerID]; ok {
			targetBase = p.BaseURL
		}
	}

	trimmed := strings.TrimRight(strings.TrimSpace(targetBase), "/")
	trimmed = strings.TrimSuffix(trimmed, "/chat/completions")
	var targetURL string
	if strings.HasSuffix(trimmed, "/v1") {
		targetURL = trimmed + "/models"
	} else {
		targetURL = trimmed + "/v1/models"
	}

	upReq, err := http.NewRequestWithContext(r.Context(), http.MethodGet, targetURL, nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create upstream models request")
		return
	}
	upReq.Header.Set("Authorization", authHeader)
	upReq.Header.Set("Accept", "application/json")

	resp, err := h.httpClient.Do(upReq)
	if err != nil {
		writeError(w, http.StatusBadGateway, fmt.Sprintf("Upstream call error: %v", err))
		return
	}
	defer resp.Body.Close()

	for k, vv := range resp.Header {
		if !strings.EqualFold(k, "Content-Length") {
			for _, v := range vv {
				w.Header().Add(k, v)
			}
		}
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

// HandleMetrics returns operational metrics summary.
func (h *Handler) HandleMetrics(w http.ResponseWriter, r *http.Request) {
	summary := h.tracker.GetSummary()
	writeJSON(w, http.StatusOK, summary)
}

// HandlePrometheusMetrics returns standard Prometheus exposition format text for scraper integration.
func (h *Handler) HandlePrometheusMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(h.tracker.FormatPrometheus()))
}

// HandleListKeys returns all registered tenant API keys.
func (h *Handler) HandleListKeys(w http.ResponseWriter, r *http.Request) {
	if !h.requireMasterAuth(r) {
		writeError(w, http.StatusUnauthorized, "Unauthorized: Master API key required")
		return
	}
	keys := h.authStore.ListKeys()
	writeJSON(w, http.StatusOK, keys)
}

// CreateKeyRequest represents incoming payload for new tenant key.
type CreateKeyRequest struct {
	Name string `json:"name"`
	Tier string `json:"tier"`
}

// HandleCreateKey provisions a new tenant API key.
func (h *Handler) HandleCreateKey(w http.ResponseWriter, r *http.Request) {
	if !h.requireMasterAuth(r) {
		writeError(w, http.StatusUnauthorized, "Unauthorized: Master API key required")
		return
	}
	var req CreateKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	key, err := h.authStore.CreateKey(req.Name, req.Tier)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, key)
}

// HandleDeleteKey revokes an API key.
func (h *Handler) HandleDeleteKey(w http.ResponseWriter, r *http.Request) {
	if !h.requireMasterAuth(r) {
		writeError(w, http.StatusUnauthorized, "Unauthorized: Master API key required")
		return
	}
	key := r.URL.Query().Get("key")
	if key == "" {
		writeError(w, http.StatusBadRequest, "Missing key query parameter")
		return
	}

	revoked := h.authStore.RevokeKey(key)
	writeJSON(w, http.StatusOK, map[string]any{"revoked": revoked})
}
