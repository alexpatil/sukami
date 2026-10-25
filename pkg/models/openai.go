package models

import (
	"encoding/json"
	"strings"
)

// chatAlias is the intermediate struct for custom ChatMessage JSON methods.
type chatAlias struct {
	Role             string          `json:"role"`
	Content          json.RawMessage `json:"content"`
	Name             string          `json:"name,omitempty"`
	ToolCalls        json.RawMessage `json:"tool_calls,omitempty"`
	ToolCallID       string          `json:"tool_call_id,omitempty"`
	FunctionCall     json.RawMessage `json:"function_call,omitempty"`
	ReasoningContent string          `json:"reasoning_content,omitempty"`
}

// ChatMessage represents an OpenAI chat message with multimodal content support.
// Content holds extracted text for internal processing; RawContent preserves original format.
type ChatMessage struct {
	Role             string
	Content          string          // Extracted text for pruning, caching, token counting
	Name             string
	RawContent       json.RawMessage // Original content (string or array) for upstream passthrough
	ToolCalls        json.RawMessage
	ToolCallID       string
	FunctionCall     json.RawMessage
	ReasoningContent string
}

// UnmarshalJSON handles content as either a plain string or multimodal content array.
func (m *ChatMessage) UnmarshalJSON(data []byte) error {
	var aux chatAlias
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	m.Role = aux.Role
	m.Name = aux.Name
	m.RawContent = aux.Content
	m.ToolCalls = aux.ToolCalls
	m.ToolCallID = aux.ToolCallID
	m.FunctionCall = aux.FunctionCall
	m.ReasoningContent = aux.ReasoningContent
	m.Content = extractTextContent(aux.Content)
	return nil
}

// MarshalJSON outputs content in its original format for upstream passthrough.
func (m ChatMessage) MarshalJSON() ([]byte, error) {
	out := chatAlias{
		Role:             m.Role,
		Name:             m.Name,
		ToolCalls:        m.ToolCalls,
		ToolCallID:       m.ToolCallID,
		FunctionCall:     m.FunctionCall,
		ReasoningContent: m.ReasoningContent,
	}
	if m.RawContent != nil {
		out.Content = m.RawContent
	} else {
		out.Content, _ = json.Marshal(m.Content)
	}
	return json.Marshal(out)
}

// extractTextContent extracts text from a string or multimodal content array.
func extractTextContent(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	var parts []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if json.Unmarshal(raw, &parts) == nil {
		var texts []string
		for _, p := range parts {
			if p.Type == "text" && p.Text != "" {
				texts = append(texts, p.Text)
			}
		}
		return strings.Join(texts, "\n")
	}
	return ""
}

// IsTextContent returns true if raw content is a JSON string (not a multimodal array).
func IsTextContent(raw json.RawMessage) bool {
	return len(raw) == 0 || raw[0] == '"'
}

// ChatCompletionRequest represents an OpenAI chat completion request with full field passthrough.
type ChatCompletionRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Stream      bool          `json:"stream,omitempty"`
	Temperature *float64      `json:"temperature,omitempty"`
	MaxTokens   *int          `json:"max_tokens,omitempty"`
	TopP        *float64      `json:"top_p,omitempty"`
	N           *int          `json:"n,omitempty"`
	Stop        any           `json:"stop,omitempty"`
	User        string        `json:"user,omitempty"`
	// Passthrough fields for full provider compatibility
	Tools          json.RawMessage `json:"tools,omitempty"`
	ToolChoice     json.RawMessage `json:"tool_choice,omitempty"`
	ResponseFormat json.RawMessage `json:"response_format,omitempty"`
	Seed           *int            `json:"seed,omitempty"`
	FreqPenalty    *float64        `json:"frequency_penalty,omitempty"`
	PresPenalty    *float64        `json:"presence_penalty,omitempty"`
	LogProbs       *bool           `json:"logprobs,omitempty"`
	TopLogProbs    *int            `json:"top_logprobs,omitempty"`
	StreamOptions  json.RawMessage `json:"stream_options,omitempty"`
}

// ChatChoice represents a choice returned by OpenAI.
type ChatChoice struct {
	Index        int         `json:"index"`
	Message      ChatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

// StreamDelta represents delta content inside an SSE streaming chunk.
type StreamDelta struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}

// StreamChoice represents streaming choice.
type StreamChoice struct {
	Index        int         `json:"index"`
	Delta        StreamDelta `json:"delta"`
	FinishReason *string     `json:"finish_reason"`
}

// Usage captures token counts.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ChatCompletionResponse represents standard OpenAI chat completion response.
type ChatCompletionResponse struct {
	ID      string       `json:"id"`
	Object  string       `json:"object"`
	Created int64        `json:"created"`
	Model   string       `json:"model"`
	Choices []ChatChoice `json:"choices"`
	Usage   Usage        `json:"usage"`
}

// ChatCompletionChunk represents standard OpenAI SSE streaming chunk.
type ChatCompletionChunk struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Created int64          `json:"created"`
	Model   string         `json:"model"`
	Choices []StreamChoice `json:"choices"`
	Usage   *Usage         `json:"usage,omitempty"`
}

// RawPayload can parse generic JSON if needed.
type RawPayload map[string]json.RawMessage
