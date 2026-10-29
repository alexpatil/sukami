package proxy

import (
	"encoding/json"
	"net/http"
)

// HandleModels returns standard OpenAI-compatible models list.
func (h *Handler) HandleModels(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"object": "list",
		"data": []map[string]any{
			{"id": "gpt-4o", "object": "model", "owned_by": "openai"},
			{"id": "gpt-4o-mini", "object": "model", "owned_by": "openai"},
			{"id": "gpt-4-turbo", "object": "model", "owned_by": "openai"},
			{"id": "claude-3-5-sonnet", "object": "model", "owned_by": "anthropic"},
			{"id": "gemini-1.5-flash", "object": "model", "owned_by": "google"},
		},
	})
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
