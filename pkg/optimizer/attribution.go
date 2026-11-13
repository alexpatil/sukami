package optimizer

import (
	"encoding/json"
	"sukami/pkg/models"
)

// InputParts is an estimate of how a request's input tokens split by purpose.
type InputParts struct {
	System  int // system messages
	History int // every earlier turn: prior user, assistant and tool messages
	Prompt  int // the last user message
	Tools   int // tool definitions sent with the request
}

// AttributeInput estimates where a request's input tokens go, using the same tokenizer as the pruner.
// In a multi-turn chat the history is resent on every turn, which is why it is counted on its own.
func AttributeInput(model string, messages []models.ChatMessage, tools json.RawMessage) InputParts {
	lastUser := -1
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			lastUser = i
			break
		}
	}

	var parts InputParts
	for i, msg := range messages {
		tokens := CountExactTokens(model, msg.Content)
		switch {
		case msg.Role == "system":
			parts.System += tokens
		case i == lastUser:
			parts.Prompt += tokens
		default:
			parts.History += tokens
		}
	}
	parts.Tools = CountExactTokens(model, string(tools))
	return parts
}
