package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sukami/pkg/models"
	"sync/atomic"
	"testing"
)

const streamedModel = "upstream-reported-model"

// echoUpstream answers with the Authorization header and system prompt it received,
// so a response served to the wrong caller or context is visible in the body.
func echoUpstream(calls *int64) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(calls, 1)
		b, _ := io.ReadAll(r.Body)
		var req models.ChatCompletionRequest
		_ = json.Unmarshal(b, &req)
		sys := ""
		for _, m := range req.Messages {
			if m.Role == "system" {
				sys = m.Content
			}
		}
		content := fmt.Sprintf("auth=%s sys=%s", r.Header.Get("Authorization"), sys)
		if req.Stream {
			w.Header().Set("Content-Type", "text/event-stream")
			fmt.Fprintf(w, "data: {\"id\":\"x\",\"model\":%q,\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":%q}}]}\n\n", streamedModel, content)
			fmt.Fprint(w, "data: [DONE]\n\n")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"id":"x","object":"chat.completion","model":%q,"choices":[{"index":0,"message":{"role":"assistant","content":%q},"finish_reason":"stop"}],"usage":{"prompt_tokens":5,"completion_tokens":5,"total_tokens":10}}`, req.Model, content)
	}))
}

type isoReq struct {
	auth    string
	headers map[string]string
	stream  bool
	msgs    []models.ChatMessage
}

func (q isoReq) send(h *Handler) *httptest.ResponseRecorder {
	body, _ := json.Marshal(models.ChatCompletionRequest{Model: "gpt-4o", Messages: q.msgs, Stream: q.stream})
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	if q.auth != "" {
		req.Header.Set("Authorization", "Bearer "+q.auth)
	}
	for k, v := range q.headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.HandleChatCompletions(rec, req)
	return rec
}

func userMsg(content string) models.ChatMessage {
	return models.ChatMessage{Role: "user", Content: content}
}

func TestCache_IsolatedPerUpstreamKey(t *testing.T) {
	var calls int64
	up := echoUpstream(&calls)
	defer up.Close()
	h, _ := createTestStreamHandler(up.URL)

	q := []models.ChatMessage{userMsg("Summarize our Q3 internal revenue numbers for the board")}
	isoReq{auth: "sk-userA-secret", msgs: q}.send(h)
	b := isoReq{auth: "sk-userB-secret", msgs: q}.send(h).Body.String()

	if strings.Contains(b, "sk-userA-secret") || calls != 2 {
		t.Errorf("user B was served user A's cached response (upstream calls=%d): %s", calls, b)
	}
}

func TestCache_SimilarityRespectsSystemPrompt(t *testing.T) {
	var calls int64
	up := echoUpstream(&calls)
	defer up.Close()
	h, _ := createTestStreamHandler(up.URL)

	ask := userMsg("Explain how database indexing works in detail")
	isoReq{auth: "sk-same", msgs: []models.ChatMessage{{Role: "system", Content: "Always reply in French."}, ask}}.send(h)
	second := isoReq{auth: "sk-same", msgs: []models.ChatMessage{{Role: "system", Content: "Always reply in Hindi."}, ask}}.send(h).Body.String()

	if strings.Contains(second, "French") || calls != 2 {
		t.Errorf("Hindi request was served the cached French answer (upstream calls=%d): %s", calls, second)
	}
}

func TestCache_SameCallerSameContextStillHits(t *testing.T) {
	var calls int64
	up := echoUpstream(&calls)
	defer up.Close()
	h, _ := createTestStreamHandler(up.URL)

	sys := models.ChatMessage{Role: "system", Content: "You are a helpful assistant."}
	for _, p := range []string{
		"Explain how database indexing works in detail",
		"explain how database indexing works in detail?",
		"Explain in detail how database indexing works",
	} {
		isoReq{auth: "sk-same", msgs: []models.ChatMessage{sys, userMsg(p)}}.send(h)
	}

	if calls != 1 {
		t.Errorf("expected 1 upstream call (rest served from cache), got %d", calls)
	}
}

func TestCache_TenantScopedPerUpstreamTarget(t *testing.T) {
	var callsA, callsB int64
	upA := echoUpstream(&callsA)
	defer upA.Close()
	upB := echoUpstream(&callsB)
	defer upB.Close()
	h, _ := createTestStreamHandler(upA.URL)
	tenant, err := h.authStore.CreateKey("TeamAlpha", "pro")
	if err != nil {
		t.Fatalf("failed to create tenant key: %v", err)
	}

	q := []models.ChatMessage{userMsg("What is a B-tree index used for")}
	headers := func(target string) map[string]string {
		return map[string]string{"X-Sukami-Key": tenant.Key, "X-Upstream-Key": "provider-key", "X-Upstream-Url": target}
	}
	isoReq{headers: headers(upA.URL), msgs: q}.send(h)
	isoReq{headers: headers(upB.URL), msgs: q}.send(h)

	if callsA != 1 || callsB != 1 {
		t.Errorf("each upstream target must be called once, got A=%d B=%d", callsA, callsB)
	}
}

func TestCache_StreamCapturedResponseReportsUpstreamModel(t *testing.T) {
	var calls int64
	up := echoUpstream(&calls)
	defer up.Close()
	h, _ := createTestStreamHandler(up.URL)

	q := []models.ChatMessage{userMsg("What is a B-tree index used for")}
	isoReq{auth: "sk-same", stream: true, msgs: q}.send(h)
	rec := isoReq{auth: "sk-same", msgs: q}.send(h)

	if rec.Header().Get("X-Cache") != "HIT" {
		t.Fatalf("expected the captured stream to be served from cache, got X-Cache=%q", rec.Header().Get("X-Cache"))
	}
	var resp models.ChatCompletionResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("cached body is not a chat completion: %v", err)
	}
	if resp.Model != streamedModel {
		t.Errorf("cached response model = %q, want the upstream-reported %q", resp.Model, streamedModel)
	}
}

func TestCache_KeyIncludesRequestParams(t *testing.T) {
	var calls int64
	up := echoUpstream(&calls)
	defer up.Close()
	h, _ := createTestStreamHandler(up.URL)

	post := func(extra string) string {
		body := `{"model":"gpt-4o","messages":[{"role":"user","content":"Describe the invoice in this request"}]` + extra + `}`
		req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer sk-same")
		rec := httptest.NewRecorder()
		h.HandleChatCompletions(rec, req)
		return rec.Header().Get("X-Cache")
	}

	post("")
	for _, extra := range []string{
		`,"response_format":{"type":"json_object"}`,
		`,"max_tokens":5`,
		`,"temperature":0`,
		`,"tools":[{"type":"function","function":{"name":"lookup"}}]`,
	} {
		if got := post(extra); got != "MISS" {
			t.Errorf("request with %s must not reuse the plain request's entry, got X-Cache=%q", extra, got)
		}
	}

	// Delivery options do not change the completion, so they still share the entry.
	if got := post(`,"user":"u-1"`); got != "HIT" {
		t.Errorf("request differing only in user must be served from cache, got X-Cache=%q", got)
	}
	if got := post(`,"max_tokens":5`); got != "HIT" {
		t.Errorf("repeating identical parameters must be served from cache, got X-Cache=%q", got)
	}
}
