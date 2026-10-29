package proxy

import (
	"net/http"
	"strings"
)

// DefaultUpstreamURL is the standard fallback completions base endpoint when no upstream flag or custom routing is specified.
const DefaultUpstreamURL = "https://api.openai.com/v1"

// resolveUpstreamTarget dynamically determines the target completions URL.
// Priority:
// 1. Explicit request header: X-Upstream-Url or X-Base-Url
// 2. Server default upstreamURL
func (h *Handler) resolveUpstreamTarget(r *http.Request) string {
	// 1. Explicit custom URL headers
	if custom := r.Header.Get("X-Upstream-Url"); custom != "" {
		return buildCompletionsURL(custom)
	}
	if custom := r.Header.Get("X-Base-Url"); custom != "" {
		return buildCompletionsURL(custom)
	}

	// 2. Default server upstream
	return buildCompletionsURL(h.upstreamURL)
}

func buildCompletionsURL(base string) string {
	trimmed := strings.TrimRight(strings.TrimSpace(base), "/")
	if strings.HasSuffix(trimmed, "/chat/completions") {
		return trimmed
	}
	if strings.HasSuffix(trimmed, "/v1") {
		return trimmed + "/chat/completions"
	}
	return trimmed + "/v1/chat/completions"
}
