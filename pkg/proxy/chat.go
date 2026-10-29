package proxy

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sukami/pkg/cache"
	"sukami/pkg/metrics"
	"sukami/pkg/models"
	"sukami/pkg/optimizer"
	"time"
)

// HandleChatCompletions processes OpenAI /v1/chat/completions requests.
func (h *Handler) HandleChatCompletions(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	reqID := generateReqID()

	// Limit request body to 10MB to protect against memory-exhaustion DoS
	bodyBytes, err := io.ReadAll(io.LimitReader(r.Body, 10<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Failed to read request body")
		return
	}
	if len(bodyBytes) >= 10<<20 {
		writeError(w, http.StatusRequestEntityTooLarge, "Request payload exceeds 10MB limit")
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

	// 1. Gateway & Tenant Authorization Check (Cloud Security Mode)
	authorized, tenant := h.validateGatewayAuth(r)
	if !authorized {
		writeError(w, http.StatusUnauthorized, "Unauthorized: Invalid or missing gateway API key")
		return
	}
	if tenant != nil {
		if !h.limiter.Allow(tenant.Key, tenant.RateLimitPerMin) {
			w.Header().Set("Retry-After", "60")
			writeError(w, http.StatusTooManyRequests, fmt.Sprintf("Rate limit exceeded (%d req/min)", tenant.RateLimitPerMin))
			return
		}
	}

	// 2. Resolve Upstream Authorization BEFORE Cache Check (Strict Zero-Cache-Leak)
	authHeader := h.resolveUpstreamAuth(r)
	if authHeader == "" && os.Getenv("SUKAMI_DEMO_MODE") != "true" {
		writeError(w, http.StatusUnauthorized, "Missing API key. Provide Authorization: Bearer <key> in request header or set UPSTREAM_API_KEY environment variable.")
		return
	}

	clientKey := resolveSukamiKey(r)
	userPrompt := extractUserPrompt(chatReq.Messages)

	// Smart Prompt Complexity Model Router:
	// Automatically downgrade simple queries to economy tier (e.g. gpt-4o-mini) when enabled,
	// saving up to 95% in token costs, while preserving flagship models for complex code & math.
	if r.Header.Get("X-Sukami-Smart-Route") == "true" || r.Header.Get("X-Sukami-Smart-Route") == "auto" || os.Getenv("ENABLE_SMART_ROUTING") == "true" {
		classification := optimizer.ClassifyPromptComplexity(userPrompt)
		targetURL := h.resolveUpstreamTarget(r)
		if routedModel, changed, reason := optimizer.ResolveSmartModel(targetURL, chatReq.Model, classification.Tier); changed {
			w.Header().Set("X-Sukami-Original-Model", chatReq.Model)
			w.Header().Set("X-Sukami-Routed-Model", routedModel)
			w.Header().Set("X-Sukami-Route-Tier", string(classification.Tier))
			w.Header().Set("X-Sukami-Route-Reason", reason)
			chatReq.Model = routedModel
		}
	}

	// 3. Optimize input messages using model-specific token rules
	comp := optimizer.OptimizeMessagesForModel(chatReq.Model, chatReq.Messages)
	if clientKey != "" {
		h.authStore.RecordTokenUsage(clientKey, comp.OptimizedTokens, comp.SavedTokens)
	}

	// Scope cache by tenant to prevent cross-tenant data leaks (M-6)
	cacheModel := scopedCacheModel(clientKey, chatReq.Model)

	// Build deterministic hash key for semantic cache
	cacheKey := cache.ComputeHash(cacheModel, buildPromptFingerprint(comp.Messages))

	// 4. Check Semantic Cache (Exact Hash or Cosine Similarity > 0.85)
	if cachedBytes, hit, sim := h.cache.GetSemantic(cacheModel, userPrompt, 0.85); hit {
		h.respondFromCache(w, r, cachedBytes, chatReq, comp, reqID, userPrompt, sim, start)
		return
	}

	// 5. Cache Miss: apply output density optimization if opted-in
	if r.Header.Get("X-Sukami-Density") == "true" || os.Getenv("ENABLE_OUTPUT_DENSITY") == "true" {
		optMessages, _ := optimizer.ApplyOutputOptimization(comp.Messages)
		chatReq.Messages = optMessages
	}

	if authHeader == "" && os.Getenv("SUKAMI_DEMO_MODE") == "true" {
		h.respondSimulated(w, r, chatReq, comp, cacheKey, cacheModel, reqID, userPrompt, start)
		return
	}

	// 6. Forward upstream to resolved provider target
	h.forwardUpstream(w, r, chatReq, comp, cacheKey, cacheModel, reqID, userPrompt, authHeader, start)
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
	h.recordHitMetric(reqID, chatReq.Model, userPrompt, comp, start, statusText, cachedBytes)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Cache", "HIT")
	w.Header().Set("X-Sukami-Latency", fmt.Sprintf("%dms", latency))
	w.Header().Set("X-Sukami-Savings", fmt.Sprintf("%.1f%%", comp.SavingsPercent))
	w.Header().Set("X-Sukami-Pruned-Tokens", fmt.Sprintf("%d", comp.SavedTokens))
	w.Header().Set("X-Sukami-Request-ID", reqID)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(cachedBytes)
}

func (h *Handler) recordHitMetric(id, model, prompt string, comp optimizer.CompressionResult, start time.Time, status string, cachedBytes []byte) {
	var cachedResp struct {
		Usage struct {
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
	}
	outputSavedTokens := 0
	if json.Unmarshal(cachedBytes, &cachedResp) == nil && cachedResp.Usage.CompletionTokens > 0 {
		outputSavedTokens = cachedResp.Usage.CompletionTokens
	}

	record := metrics.RequestRecord{
		ID:                id,
		Timestamp:         time.Now(),
		Model:             model,
		PromptPreview:     truncate(prompt, 60),
		OriginalTokens:    comp.OriginalTokens + outputSavedTokens,
		SavedTokens:       comp.OriginalTokens + outputSavedTokens,
		InputSavedTokens:  comp.OriginalTokens,
		OutputSavedTokens: outputSavedTokens,
		BilledTokens:      0,
		SavingsPercent:    100.0,
		CacheHit:          true,
		LatencyMs:         time.Since(start).Milliseconds(),
		Status:            status,
	}

	costUSD, _ := metrics.CalculateSavingsUSD(model, record.InputSavedTokens, record.OutputSavedTokens)
	record.CostSavedUSD = costUSD
	h.tracker.RecordRequest(record)
}
