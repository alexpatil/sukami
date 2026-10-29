package proxy

import (
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
	}
	if targetBase == "" {
		writeError(w, http.StatusBadRequest, "No upstream URL configured: provide X-Upstream-Url header or configure UPSTREAM_URL")
		return
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
