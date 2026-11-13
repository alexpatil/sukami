package optimizer

import (
	"encoding/json"
	"sukami/pkg/models"
	"testing"
)

func TestAttributeInput_SplitsByPurpose(t *testing.T) {
	const model = "any-model"
	system := "You are a support assistant for an electronics store."
	first := "Does the warranty cover water damage?"
	reply := "No. Water damage is excluded under clause four."
	toolResult := "order status: shipped"
	last := "And accidental drops?"
	tools := json.RawMessage(`[{"type":"function","function":{"name":"lookup_order"}}]`)

	got := AttributeInput(model, []models.ChatMessage{
		{Role: "system", Content: system},
		{Role: "user", Content: first},
		{Role: "assistant", Content: reply},
		{Role: "tool", Content: toolResult},
		{Role: "user", Content: last},
	}, tools)

	want := InputParts{
		System:  CountExactTokens(model, system),
		History: CountExactTokens(model, first) + CountExactTokens(model, reply) + CountExactTokens(model, toolResult),
		Prompt:  CountExactTokens(model, last),
		Tools:   CountExactTokens(model, string(tools)),
	}
	if got != want {
		t.Errorf("AttributeInput = %+v, want %+v", got, want)
	}
	if got.System == 0 || got.History == 0 || got.Prompt == 0 || got.Tools == 0 {
		t.Errorf("every part should be counted, got %+v", got)
	}
}

func TestAttributeInput_WithoutUserMessageCountsNoPrompt(t *testing.T) {
	got := AttributeInput("any-model", []models.ChatMessage{
		{Role: "system", Content: "Be brief."},
		{Role: "assistant", Content: "Hello, how can I help?"},
	}, nil)

	if got.Prompt != 0 || got.Tools != 0 {
		t.Errorf("expected no prompt or tool tokens, got %+v", got)
	}
	if got.System == 0 || got.History == 0 {
		t.Errorf("system and history should be counted, got %+v", got)
	}
}
