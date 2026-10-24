package proxy

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sukami/pkg/models"
	"sukami/pkg/optimizer"
	"time"
)

// respondSimulated sends an intelligent simulated response when no API key is provided.
func (h *Handler) respondSimulated(
	w http.ResponseWriter,
	r *http.Request,
	chatReq models.ChatCompletionRequest,
	comp optimizer.CompressionResult,
	cacheKey string,
	reqID string,
	userPrompt string,
	start time.Time,
) {
	text := fmt.Sprintf(
		"Hello from Sukami Proxy! ⚡\n\nYour prompt was processed through Sukami Token Optimizer.\n• Original Tokens: %d\n• Optimized Tokens: %d\n• Tokens Saved: %d (%.1f%%)\n\nTo forward live requests upstream, provide Authorization: Bearer <key>.",
		comp.OriginalTokens, comp.OptimizedTokens, comp.SavedTokens, comp.SavingsPercent,
	)

	respObj := models.ChatCompletionResponse{
		ID:      "chatcmpl-" + reqID,
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   chatReq.Model,
		Choices: []models.ChatChoice{
			{
				Index:        0,
				Message:      models.ChatMessage{Role: "assistant", Content: text},
				FinishReason: "stop",
			},
		},
		Usage: models.Usage{
			PromptTokens:     comp.OptimizedTokens,
			CompletionTokens: optimizer.EstimateTokens(text),
			TotalTokens:      comp.OptimizedTokens + optimizer.EstimateTokens(text),
		},
	}

	respJSON, _ := json.Marshal(respObj)
	h.cache.SetSemantic(cacheKey, userPrompt, respJSON, chatReq.Model)
	h.recordMissMetric(reqID, chatReq.Model, userPrompt, comp, start, "200 OK (Simulated)")

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Cache", "MISS")
	w.Header().Set("X-Sukami-Pruned-Tokens", fmt.Sprintf("%d", comp.SavedTokens))

	if chatReq.Stream {
		h.streamCachedResponse(w, r, respJSON, chatReq.Model)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(respJSON)
}

// streamCachedResponse streams cached content over Server-Sent Events (SSE).
func (h *Handler) streamCachedResponse(w http.ResponseWriter, r *http.Request, jsonBytes []byte, model string) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonBytes)
		return
	}

	var resp models.ChatCompletionResponse
	if err := json.Unmarshal(jsonBytes, &resp); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonBytes)
		return
	}

	content := ""
	if len(resp.Choices) > 0 {
		content = resp.Choices[0].Message.Content
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	words := strings.Fields(content)
	chunkID := fmt.Sprintf("chatcmpl-stream-%d", time.Now().UnixNano())
	ctx := r.Context()

	for i, word := range words {
		select {
		case <-ctx.Done():
			return // Client disconnected: immediately abort to save bandwidth
		default:
		}

		suffix := " "
		if i == len(words)-1 {
			suffix = ""
		}

		chunk := models.ChatCompletionChunk{
			ID:      chunkID,
			Object:  "chat.completion.chunk",
			Created: time.Now().Unix(),
			Model:   model,
			Choices: []models.StreamChoice{
				{
					Index: 0,
					Delta: models.StreamDelta{Content: word + suffix},
				},
			},
		}

		b, _ := json.Marshal(chunk)
		fmt.Fprintf(w, "data: %s\n\n", b)
		flusher.Flush()
		time.Sleep(15 * time.Millisecond)
	}

	fmt.Fprintf(w, "data: [DONE]\n\n")
	flusher.Flush()
}
