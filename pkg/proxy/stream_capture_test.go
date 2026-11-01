package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sukami/pkg/auth"
	"sukami/pkg/cache"
	"sukami/pkg/metrics"
	"sukami/pkg/models"
	"testing"
	"time"
)

func createTestStreamHandler(upstreamURL string) (*Handler, *cache.MemoryCache) {
	c := cache.NewMemoryCache(1 * time.Hour)
	tracker := metrics.NewTracker(10)
	ks := auth.NewMemoryKeyStore()
	limiter := auth.NewRateLimiter()
	h := NewHandler(c, tracker, ks, limiter, upstreamURL)
	return h, c
}

func TestStreamCapture_SuccessfulSynthesisAndCacheHit(t *testing.T) {
	mockUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.WriteHeader(http.StatusOK)

		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("expected flusher")
		}

		chunks := []string{
			`{"id":"chatcmpl-stream1","object":"chat.completion.chunk","created":1730000000,"model":"gpt-4o","choices":[{"index":0,"delta":{"role":"assistant"},"finish_reason":null}]}`,
			`{"id":"chatcmpl-stream1","object":"chat.completion.chunk","created":1730000000,"model":"gpt-4o","choices":[{"index":0,"delta":{"content":"Concurrency in "},"finish_reason":null}]}`,
			`{"id":"chatcmpl-stream1","object":"chat.completion.chunk","created":1730000000,"model":"gpt-4o","choices":[{"index":0,"delta":{"content":"Go is awesome."},"finish_reason":null}]}`,
			`{"id":"chatcmpl-stream1","object":"chat.completion.chunk","created":1730000000,"model":"gpt-4o","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`,
		}

		for _, chunk := range chunks {
			fmt.Fprintf(w, "data: %s\n\n", chunk)
			flusher.Flush()
			time.Sleep(2 * time.Millisecond)
		}
		fmt.Fprintf(w, "data: [DONE]\n\n")
		flusher.Flush()
	}))
	defer mockUpstream.Close()

	handler, _ := createTestStreamHandler(mockUpstream.URL)

	userPrompt := "Tell me about Go concurrency"
	reqPayload := models.ChatCompletionRequest{
		Model: "gpt-4o",
		Messages: []models.ChatMessage{
			{Role: "user", Content: userPrompt},
		},
		Stream: true,
	}
	bodyBytes, _ := json.Marshal(reqPayload)

	// First Request: Cache MISS -> Streamed from upstream & captured into cache
	req1 := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(bodyBytes))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Authorization", "Bearer sk-test-key")
	rec1 := httptest.NewRecorder()

	handler.HandleChatCompletions(rec1, req1)

	if rec1.Code != http.StatusOK {
		t.Fatalf("first request expected status 200, got %d. Body: %s", rec1.Code, rec1.Body.String())
	}
	if xCache := rec1.Header().Get("X-Cache"); xCache != "MISS" {
		t.Errorf("expected X-Cache MISS on first request, got %q", xCache)
	}

	body1 := rec1.Body.String()
	if !strings.Contains(body1, "Concurrency in ") || !strings.Contains(body1, "Go is awesome.") {
		t.Fatalf("client did not receive full streamed chunks: %s", body1)
	}
	if !strings.Contains(body1, "data: [DONE]") {
		t.Fatalf("client did not receive [DONE] terminator: %s", body1)
	}

	// Second Request (Non-streaming): Must be instant Cache HIT
	reqPayload.Stream = false
	bodyBytes2, _ := json.Marshal(reqPayload)
	req2 := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(bodyBytes2))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer sk-test-key")
	rec2 := httptest.NewRecorder()

	handler.HandleChatCompletions(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Fatalf("second request expected status 200, got %d. Body: %s", rec2.Code, rec2.Body.String())
	}
	if xCache := rec2.Header().Get("X-Cache"); xCache != "HIT" {
		t.Errorf("expected X-Cache HIT on second request, got %q", xCache)
	}

	var cachedObj models.ChatCompletionResponse
	if err := json.Unmarshal(rec2.Body.Bytes(), &cachedObj); err != nil {
		t.Fatalf("second request response is not valid ChatCompletionResponse JSON: %v", err)
	}
	expectedText := "Concurrency in Go is awesome."
	if len(cachedObj.Choices) == 0 || cachedObj.Choices[0].Message.Content != expectedText {
		t.Errorf("cached message content = %q; want %q", cachedObj.Choices[0].Message.Content, expectedText)
	}
	if cachedObj.Choices[0].FinishReason != "stop" {
		t.Errorf("cached finish reason = %q; want 'stop'", cachedObj.Choices[0].FinishReason)
	}

	// Third Request (Streaming again): Must stream from cache as HIT
	reqPayload.Stream = true
	bodyBytes3, _ := json.Marshal(reqPayload)
	req3 := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(bodyBytes3))
	req3.Header.Set("Content-Type", "application/json")
	req3.Header.Set("Authorization", "Bearer sk-test-key")
	rec3 := httptest.NewRecorder()

	handler.HandleChatCompletions(rec3, req3)

	if rec3.Code != http.StatusOK {
		t.Fatalf("third request expected status 200, got %d", rec3.Code)
	}
	if xCache := rec3.Header().Get("X-Cache"); xCache != "HIT" {
		t.Errorf("expected X-Cache HIT on third streaming request, got %q", xCache)
	}
	if !strings.Contains(rec3.Body.String(), "Concurrency ") || !strings.Contains(rec3.Body.String(), "awesome.") {
		t.Errorf("expected cached stream content in response body, got: %s", rec3.Body.String())
	}
}

func TestStreamCapture_UpstreamErrorNotCached(t *testing.T) {
	mockErrUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"upstream rate limit"}`, http.StatusTooManyRequests)
	}))
	defer mockErrUpstream.Close()

	handler, memCache := createTestStreamHandler(mockErrUpstream.URL)

	reqPayload := models.ChatCompletionRequest{
		Model: "gpt-4o",
		Messages: []models.ChatMessage{
			{Role: "user", Content: "Failing query"},
		},
		Stream: true,
	}
	bodyBytes, _ := json.Marshal(reqPayload)

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer sk-test-key")
	rec := httptest.NewRecorder()

	handler.HandleChatCompletions(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429 status, got %d", rec.Code)
	}

	_, hit, _ := memCache.GetSemantic("gpt-4o", "Failing query", 0.85)
	if hit {
		t.Errorf("failed upstream stream must not be cached")
	}
}

type brokenWriter struct {
	http.ResponseWriter
	failAfter int
	written   int
}

func (b *brokenWriter) Write(p []byte) (int, error) {
	b.written += len(p)
	if b.written > b.failAfter {
		return 0, io.ErrClosedPipe
	}
	return b.ResponseWriter.Write(p)
}

func (b *brokenWriter) Flush() {}

func TestStreamCapture_ClientDisconnectPremature(t *testing.T) {
	mockUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		for i := 0; i < 5; i++ {
			fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"content\":\"chunk%d \"}}]}\n\n", i)
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
			time.Sleep(5 * time.Millisecond)
		}
	}))
	defer mockUpstream.Close()

	handler, memCache := createTestStreamHandler(mockUpstream.URL)

	reqPayload := models.ChatCompletionRequest{
		Model: "gpt-4o",
		Messages: []models.ChatMessage{
			{Role: "user", Content: "Disconnected prompt"},
		},
		Stream: true,
	}
	bodyBytes, _ := json.Marshal(reqPayload)

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer sk-test-key")
	rec := httptest.NewRecorder()
	bw := &brokenWriter{ResponseWriter: rec, failAfter: 15}

	handler.HandleChatCompletions(bw, req)

	_, hit, _ := memCache.GetSemantic("gpt-4o", "Disconnected prompt", 0.85)
	if hit {
		t.Errorf("incomplete stream from broken connection must NOT be cached")
	}
}
