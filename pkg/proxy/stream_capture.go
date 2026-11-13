package proxy

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sukami/pkg/models"
	"sukami/pkg/optimizer"
	"time"
)

// streamAccumulator assembles in-flight SSE stream chunks into a synthesizable response.
type streamAccumulator struct {
	id           string
	model        string
	role         string
	contentBuf   strings.Builder
	finishReason string
	usage        *models.Usage
	completed    bool
}

func newStreamAccumulator() *streamAccumulator {
	return &streamAccumulator{}
}

func (a *streamAccumulator) processLine(line []byte) {
	trimmed := bytes.TrimSpace(line)
	if !bytes.HasPrefix(trimmed, []byte("data:")) {
		return
	}
	payload := bytes.TrimSpace(bytes.TrimPrefix(trimmed, []byte("data:")))
	if len(payload) == 0 {
		return
	}
	if bytes.Equal(payload, []byte("[DONE]")) {
		a.completed = true
		return
	}

	var chunk models.ChatCompletionChunk
	if err := json.Unmarshal(payload, &chunk); err != nil {
		return
	}

	if a.id == "" && chunk.ID != "" {
		a.id = chunk.ID
	}
	if chunk.Model != "" {
		a.model = chunk.Model
	}
	if chunk.Usage != nil {
		a.usage = chunk.Usage
	}

	for _, ch := range chunk.Choices {
		if ch.Delta.Role != "" {
			a.role = ch.Delta.Role
		}
		if ch.Delta.Content != "" {
			a.contentBuf.WriteString(ch.Delta.Content)
		}
		if ch.FinishReason != nil && *ch.FinishReason != "" {
			a.finishReason = *ch.FinishReason
		}
	}
}

func (a *streamAccumulator) synthesizeResponse(model, reqID string, optTokens int) []byte {
	// Only cache if content was accumulated and stream completed cleanly
	if a.contentBuf.Len() == 0 || (!a.completed && a.finishReason == "") {
		return nil
	}

	fullText := a.contentBuf.String()
	role := a.role
	if role == "" {
		role = "assistant"
	}
	finishReason := a.finishReason
	if finishReason == "" {
		finishReason = "stop"
	}
	id := a.id
	if id == "" {
		id = "chatcmpl-" + reqID
	}

	promptTokens := optTokens
	completionTokens := optimizer.EstimateTokens(fullText)
	totalTokens := promptTokens + completionTokens
	if a.usage != nil && a.usage.TotalTokens > 0 {
		promptTokens = a.usage.PromptTokens
		completionTokens = a.usage.CompletionTokens
		totalTokens = a.usage.TotalTokens
	}

	resp := models.ChatCompletionResponse{
		ID:      id,
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   model,
		Choices: []models.ChatChoice{
			{
				Index: 0,
				Message: models.ChatMessage{
					Role:    role,
					Content: fullText,
				},
				FinishReason: finishReason,
			},
		},
		Usage: models.Usage{
			PromptTokens:     promptTokens,
			CompletionTokens: completionTokens,
			TotalTokens:      totalTokens,
		},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		return nil
	}
	return data
}

// forwardAndCaptureStream streams SSE chunks to the client as they arrive,
// while accumulating delta chunks to cache successful completions.
func (h *Handler) forwardAndCaptureStream(
	w http.ResponseWriter,
	resp *http.Response,
	chatReq models.ChatCompletionRequest,
	comp optimizer.CompressionResult,
	cacheKey string,
	cacheModel string,
	reqID string,
	userPrompt string,
	start time.Time,
) {
	flusher, isFlusher := w.(http.Flusher)
	reader := bufio.NewReader(resp.Body)
	acc := newStreamAccumulator()

	for {
		line, readErr := reader.ReadBytes('\n')
		if len(line) > 0 {
			if _, writeErr := w.Write(line); writeErr != nil {
				break // Client disconnected prematurely
			}
			if isFlusher {
				flusher.Flush()
			}
			acc.processLine(line)
		}
		if readErr != nil {
			break
		}
	}

	var cachedBytes []byte
	if resp.StatusCode == http.StatusOK {
		if synthesized := acc.synthesizeResponse(cacheModel, reqID, comp.OptimizedTokens); synthesized != nil {
			h.cache.SetSemantic(cacheKey, userPrompt, synthesized, cacheModel)
			cachedBytes = synthesized
		}
	}

	h.recordMissMetric(reqID, chatReq.Model, userPrompt, comp, start, fmt.Sprintf("%d Stream", resp.StatusCode), cachedBytes)
}
