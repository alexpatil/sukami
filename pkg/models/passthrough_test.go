package models

import (
	"encoding/json"
	"testing"
)

func TestChatCompletionRequest_ForwardsUnmodeledFields(t *testing.T) {
	in := `{"model":"m","messages":[{"role":"user","content":"hi"}],"max_tokens":7,` +
		`"max_completion_tokens":50,"reasoning_effort":"low","parallel_tool_calls":false,"metadata":{"a":"b"}}`

	var req ChatCompletionRequest
	if err := json.Unmarshal([]byte(in), &req); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if req.Model != "m" || req.MaxTokens == nil || *req.MaxTokens != 7 || len(req.Messages) != 1 {
		t.Fatalf("declared fields were not decoded: %+v", req)
	}
	if _, kept := req.Extra["max_tokens"]; kept {
		t.Errorf("a declared field must not be duplicated into Extra")
	}

	out, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got map[string]json.RawMessage
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("re-decode: %v", err)
	}
	want := map[string]string{
		"model":                 `"m"`,
		"max_tokens":            `7`,
		"max_completion_tokens": `50`,
		"reasoning_effort":      `"low"`,
		"parallel_tool_calls":   `false`,
		"metadata":              `{"a":"b"}`,
	}
	for name, raw := range want {
		if string(got[name]) != raw {
			t.Errorf("field %q = %s, want %s", name, got[name], raw)
		}
	}
}

func TestChatCompletionRequest_NoExtraFieldsKeepsDeclaredEncoding(t *testing.T) {
	req := ChatCompletionRequest{Model: "m", Messages: []ChatMessage{{Role: "user", Content: "hi"}}}
	out, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if want := `{"model":"m","messages":[{"role":"user","content":"hi"}]}`; string(out) != want {
		t.Errorf("got %s, want %s", out, want)
	}
}
