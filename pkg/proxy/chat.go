package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sukami/pkg/cache"
	"sukami/pkg/metrics"
	"sukami/pkg/models"
	"sukami/pkg/optimizer"
	"time"
)

// HandleChatCompletions processes OpenAI /v1/chat/completions requests.
func (h *Handler) HandleChatCompletions(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	reqID := fmt.Sprintf("req_%d", time.Now().UnixNano())

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Failed to read request body")
		return
	}

	var chatReq models.ChatCompletionRequest
	if err := json.Unmarshal(bodyBytes, &chatReq); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	if len(chatReq.Messages) == 0 {
		writeError(w, http.StatusBadRequest, "'messages' array cannot be empty")
		return
	}

	// Multi-tenant API Key authentication and rate limiting
	clientKey := resolveSukamiKey(r)
	if clientKey != "" {
		tenant, valid := h.authStore.Validate(clientKey)
		if !valid {
			writeError(w, http.StatusUnauthorized, "Unauthorized: Invalid Sukami API Key")
			return
		}
		if !h.limiter.Allow(clientKey, tenant.RateLimitPerMin) {
			w.Header().Set("Retry-After", "60")
			writeError(w, http.StatusTooManyRequests, fmt.Sprintf("Rate limit exceeded (%d req/min)", tenant.RateLimitPerMin))
			return
		}
	}

	// 1. Optimize input messages (Prompt pruning)
	comp := optimizer.OptimizeMessages(chatReq.Messages)
	userPrompt := extractUserPrompt(chatReq.Messages)
	if clientKey != "" {
		h.authStore.RecordTokenUsage(clientKey, comp.OptimizedTokens, comp.SavedTokens)
	}

	// Route generic model name to active Groq model if targeting Groq
	if strings.Contains(h.upstreamURL, "groq.com") && (chatReq.Model == "gpt-4o" || chatReq.Model == "gpt-4" || chatReq.Model == "") {
		chatReq.Model = "openai/gpt-oss-120b"
	}

	// Build deterministic hash key for semantic cache
	cacheKey := cache.ComputeHash(chatReq.Model, buildPromptFingerprint(comp.Messages))

	// 2. Check Semantic Cache (Exact Hash or Cosine Similarity > 0.85)
	if cachedBytes, hit, sim := h.cache.GetSemantic(chatReq.Model, userPrompt, 0.85); hit {
		h.respondFromCache(w, r, cachedBytes, chatReq, comp, reqID, userPrompt, sim, start)
		return
	}

	// 3. Cache Miss: apply output density optimization
	optMessages, _ := optimizer.ApplyOutputOptimization(comp.Messages)
	chatReq.Messages = optMessages
	authHeader := resolveUpstreamAuth(r)

	// Fallback to simulated completion if no API key provided
	if authHeader == "" {
		h.respondSimulated(w, r, chatReq, comp, cacheKey, reqID, userPrompt, start)
		return
	}

	// 4. Forward upstream to OpenAI
	h.forwardUpstream(w, r, chatReq, comp, cacheKey, reqID, userPrompt, authHeader, start)
}

func (h *Handler) respondFromCache(
	w http.ResponseWriter,
	r *http.Request,
	cachedBytes []byte,
	chatReq models.ChatCompletionRequest,
	comp optimizer.CompressionResult,
	reqID string,
	userPrompt string,
	similarity float64,
	start time.Time,
) {
	latency := time.Since(start).Milliseconds()
	statusText := "200 OK (Cache)"
	if similarity < 0.999 {
		statusText = fmt.Sprintf("200 OK (Semantic %d%%)", int(similarity*100))
	}

	h.tracker.RecordRequest(metrics.RequestRecord{
		ID:                reqID,
		Timestamp:         time.Now(),
		Model:             chatReq.Model,
		PromptPreview:     truncate(userPrompt, 60),
		OriginalTokens:    comp.OriginalTokens,
		SavedTokens:       comp.OriginalTokens, // 100% saved
		InputSavedTokens:  comp.OriginalTokens,
		OutputSavedTokens: 0,
		BilledTokens:      0,
		SavingsPercent:    100.0,
		CacheHit:          true,
		LatencyMs:         latency,
		Status:            statusText,
	})

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Cache", fmt.Sprintf("HIT (%.0f%%)", similarity*100))
	w.Header().Set("X-Sukami-Saved-Tokens", fmt.Sprintf("%d", comp.OriginalTokens))

	if chatReq.Stream {
		h.streamCachedResponse(w, r, cachedBytes, chatReq.Model)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(cachedBytes)
}

func (h *Handler) forwardUpstream(
	w http.ResponseWriter,
	r *http.Request,
	chatReq models.ChatCompletionRequest,
	comp optimizer.CompressionResult,
	cacheKey string,
	reqID string,
	userPrompt string,
	authHeader string,
	start time.Time,
) {
	optimizedBody, _ := json.Marshal(chatReq)
	targetURL := h.resolveUpstreamTarget(r, chatReq.Model)

	upReq, err := http.NewRequestWithContext(r.Context(), http.MethodPost, targetURL, bytes.NewReader(optimizedBody))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create upstream request")
		return
	}

	upReq.Header.Set("Content-Type", "application/json")
	upReq.Header.Set("Authorization", authHeader)
	if strings.Contains(targetURL, "openrouter.ai") {
		upReq.Header.Set("HTTP-Referer", "https://sukami.ai")
		upReq.Header.Set("X-Title", "Sukami AI Gateway")
	}

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
		io.Copy(w, resp.Body)
		h.recordMissMetric(reqID, chatReq.Model, userPrompt, comp, start, fmt.Sprintf("%d Stream", resp.StatusCode))
		return
	}

	respBytes, err := io.ReadAll(resp.Body)
	if err == nil && resp.StatusCode == http.StatusOK {
		h.cache.SetSemantic(cacheKey, userPrompt, respBytes, chatReq.Model)
	}
	w.Write(respBytes)
	h.recordMissMetric(reqID, chatReq.Model, userPrompt, comp, start, fmt.Sprintf("%d OK", resp.StatusCode))
}

func (h *Handler) recordMissMetric(id, model, prompt string, comp optimizer.CompressionResult, start time.Time, status string) {
	h.tracker.RecordRequest(metrics.RequestRecord{
		ID:                id,
		Timestamp:         time.Now(),
		Model:             model,
		PromptPreview:     truncate(prompt, 60),
		OriginalTokens:    comp.OriginalTokens,
		SavedTokens:       comp.SavedTokens,
		InputSavedTokens:  comp.SavedTokens,
		OutputSavedTokens: 0,
		BilledTokens:      comp.OptimizedTokens,
		SavingsPercent:    comp.SavingsPercent,
		CacheHit:          false,
		LatencyMs:         time.Since(start).Milliseconds(),
		Status:            status,
	})
}
