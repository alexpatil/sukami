package proxy

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
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

	userPrompt := extractUserPrompt(chatReq.Messages)

	// Smart Prompt Complexity Model Router:
	// Automatically downgrade simple queries to economy tier (e.g. gpt-4o-mini) when enabled,
	// saving up to 95% in token costs, while preserving flagship models for complex code & math.
	if r.Header.Get("X-Sukami-Smart-Route") == "true" || r.Header.Get("X-Sukami-Smart-Route") == "auto" || os.Getenv("ENABLE_SMART_ROUTING") == "true" {
		classification := optimizer.ClassifyPromptComplexity(userPrompt)
		if routedModel, changed, reason := optimizer.ResolveSmartModel(chatReq.Model, classification.Tier); changed {
			w.Header().Set("X-Sukami-Original-Model", chatReq.Model)
			w.Header().Set("X-Sukami-Routed-Model", routedModel)
			w.Header().Set("X-Sukami-Route-Tier", string(classification.Tier))
			w.Header().Set("X-Sukami-Route-Reason", reason)
			chatReq.Model = routedModel
		}
	}

	// Smart Model Router: Map generic model names to active provider equivalents
	if strings.Contains(h.upstreamURL, "groq.com") && (chatReq.Model == "gpt-4o" || chatReq.Model == "gpt-4" || chatReq.Model == "") {
		chatReq.Model = "llama-3.3-70b-versatile"
		w.Header().Set("X-Sukami-Mapped-Model", "llama-3.3-70b-versatile")
	}

	// 1. Optimize input messages using model-specific token rules
	comp := optimizer.OptimizeMessagesForModel(chatReq.Model, chatReq.Messages)
	if clientKey != "" {
		h.authStore.RecordTokenUsage(clientKey, comp.OptimizedTokens, comp.SavedTokens)
	}

	// Scope cache by tenant to prevent cross-tenant data leaks (M-6)
	cacheModel := scopedCacheModel(clientKey, chatReq.Model)

	// Build deterministic hash key for semantic cache
	cacheKey := cache.ComputeHash(cacheModel, buildPromptFingerprint(comp.Messages))

	// 2. Check Semantic Cache (Exact Hash or Cosine Similarity > 0.85)
	if cachedBytes, hit, sim := h.cache.GetSemantic(cacheModel, userPrompt, 0.85); hit {
		h.respondFromCache(w, r, cachedBytes, chatReq, comp, reqID, userPrompt, sim, start)
		return
	}

	// 3. Cache Miss: apply output density optimization if opted-in
	if r.Header.Get("X-Sukami-Density") == "true" || os.Getenv("ENABLE_OUTPUT_DENSITY") == "true" {
		optMessages, _ := optimizer.ApplyOutputOptimization(comp.Messages)
		chatReq.Messages = optMessages
	}
	authHeader := resolveUpstreamAuth(r)

	// Production standard: require API key unless demo mode is explicitly enabled
	if authHeader == "" {
		if os.Getenv("SUKAMI_DEMO_MODE") == "true" {
			h.respondSimulated(w, r, chatReq, comp, cacheKey, cacheModel, reqID, userPrompt, start)
			return
		}
		writeError(w, http.StatusUnauthorized, "Missing API key. Provide Authorization: Bearer <key> in request header or set UPSTREAM_API_KEY environment variable.")
		return
	}

	// 4. Forward upstream to OpenAI
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

	outputSaved := 0
	var cu struct {
		Usage struct {
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
	}
	if json.Unmarshal(cachedBytes, &cu) == nil {
		outputSaved = cu.Usage.CompletionTokens
	}

	totalSaved := comp.OriginalTokens + outputSaved
	costUSD, _ := metrics.CalculateSavingsUSD(chatReq.Model, comp.OriginalTokens, outputSaved)
	h.tracker.RecordRequest(metrics.RequestRecord{
		ID:                reqID,
		Timestamp:         time.Now(),
		Model:             chatReq.Model,
		PromptPreview:     truncate(userPrompt, 60),
		OriginalTokens:    totalSaved,
		SavedTokens:       totalSaved,
		InputSavedTokens:  comp.OriginalTokens,
		OutputSavedTokens: outputSaved,
		BilledTokens:      0,
		SavingsPercent:    100.0,
		CostSavedUSD:      costUSD,
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
