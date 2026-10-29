package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sukami/pkg/models"
	"sukami/pkg/optimizer"
	"time"
)

// resolveUpstreamTarget dynamically determines the target completions URL.
// Priority:
// 1. Explicit request header: X-Upstream-Url or X-Base-Url
// 2. Server default upstreamURL
func (h *Handler) resolveUpstreamTarget(r *http.Request) string {
	if custom := r.Header.Get("X-Upstream-Url"); custom != "" {
		return buildCompletionsURL(custom)
	}
	if custom := r.Header.Get("X-Base-Url"); custom != "" {
		return buildCompletionsURL(custom)
	}
	if h.upstreamURL == "" {
		return ""
	}
	return buildCompletionsURL(h.upstreamURL)
}

func buildCompletionsURL(base string) string {
	trimmed := strings.TrimRight(strings.TrimSpace(base), "/")
	if trimmed == "" {
		return ""
	}
	if strings.HasSuffix(trimmed, "/chat/completions") {
		return trimmed
	}
	if strings.HasSuffix(trimmed, "/v1") {
		return trimmed + "/chat/completions"
	}
	return trimmed + "/v1/chat/completions"
}

func (h *Handler) forwardUpstream(
	w http.ResponseWriter,
	r *http.Request,
	chatReq models.ChatCompletionRequest,
	comp optimizer.CompressionResult,
	cacheKey string,
	cacheModel string,
	reqID string,
	userPrompt string,
	authHeader string,
	start time.Time,
) {
	targetURL := h.resolveUpstreamTarget(r)
	if targetURL == "" {
		writeError(w, http.StatusBadRequest, "No upstream URL configured: provide X-Upstream-Url header or configure UPSTREAM_URL")
		return
	}

	optimizedBody, _ := json.Marshal(chatReq)
	upReq, err := http.NewRequestWithContext(r.Context(), http.MethodPost, targetURL, bytes.NewReader(optimizedBody))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create upstream request")
		return
	}

	upReq.Header.Set("Content-Type", "application/json")
	upReq.Header.Set("Authorization", authHeader)

	resp, err := h.httpClient.Do(upReq)
	if err != nil {
		slog.Error("Upstream failure", "err", err)
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
	w.Header().Set("X-Cache", "MISS")
	w.Header().Set("X-Sukami-Pruned-Tokens", fmt.Sprintf("%d", comp.SavedTokens))
	w.WriteHeader(resp.StatusCode)

	if chatReq.Stream {
		flusher, isFlusher := w.(http.Flusher)
		buf := make([]byte, 4096)
		for {
			n, readErr := resp.Body.Read(buf)
			if n > 0 {
				if _, writeErr := w.Write(buf[:n]); writeErr != nil {
					break
				}
				if isFlusher {
					flusher.Flush()
				}
			}
			if readErr != nil {
				break
			}
		}
		h.recordMissMetric(reqID, chatReq.Model, userPrompt, comp, start, fmt.Sprintf("%d Stream", resp.StatusCode), nil)
		return
	}

	respBytes, err := io.ReadAll(resp.Body)
	if err == nil && resp.StatusCode == http.StatusOK {
		h.cache.SetSemantic(cacheKey, userPrompt, respBytes, cacheModel)
	}
	w.Write(respBytes)
	h.recordMissMetric(reqID, chatReq.Model, userPrompt, comp, start, fmt.Sprintf("%d OK", resp.StatusCode), respBytes)
}
