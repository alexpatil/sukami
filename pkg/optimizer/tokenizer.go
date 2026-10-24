package optimizer

import (
	"sync"

	"github.com/pkoukk/tiktoken-go"
	"sukami/pkg/models"
)

var (
	encCache   = make(map[string]*tiktoken.Tiktoken)
	encCacheMu sync.RWMutex
)

// getEncoding returns a cached tiktoken encoder for the given model.
// Falls back to standard "cl100k_base" if the specific model name is unknown.
func getEncoding(model string) (*tiktoken.Tiktoken, error) {
	encCacheMu.RLock()
	enc, ok := encCache[model]
	encCacheMu.RUnlock()
	if ok {
		return enc, nil
	}

	encCacheMu.Lock()
	defer encCacheMu.Unlock()
	if enc, ok := encCache[model]; ok {
		return enc, nil
	}

	var err error
	enc, err = tiktoken.EncodingForModel(model)
	if err != nil {
		enc, err = tiktoken.GetEncoding("cl100k_base")
		if err != nil {
			return nil, err
		}
	}

	encCache[model] = enc
	return enc, nil
}

// CountExactTokens calculates exact token count using OpenAI Byte-Pair Encoding (BPE).
func CountExactTokens(model, text string) int {
	if text == "" {
		return 0
	}
	enc, err := getEncoding(model)
	if err != nil {
		return (len(text) + 3) / 4
	}
	return len(enc.EncodeOrdinary(text))
}

// EncodeToTokenIDs generates actual BPE token integer IDs using tiktoken.
func EncodeToTokenIDs(model, text string) []int {
	if text == "" {
		return nil
	}
	enc, err := getEncoding(model)
	if err != nil {
		return nil
	}
	return enc.EncodeOrdinary(text)
}

// CountExactMessagesTokens computes token count for OpenAI chat messages adhering to ChatML standard.
func CountExactMessagesTokens(model string, messages []models.ChatMessage) int {
	enc, err := getEncoding(model)
	tokensPerMessage := 3
	tokensPerName := 1

	if err != nil {
		total := 3
		for _, msg := range messages {
			total += tokensPerMessage + (len(msg.Content)+3)/4 + (len(msg.Role)+3)/4
			if msg.Name != "" {
				total += (len(msg.Name)+3)/4 + tokensPerName
			}
		}
		return total
	}

	total := 0
	for _, msg := range messages {
		total += tokensPerMessage
		total += len(enc.EncodeOrdinary(msg.Content))
		total += len(enc.EncodeOrdinary(msg.Role))
		if msg.Name != "" {
			total += len(enc.EncodeOrdinary(msg.Name)) + tokensPerName
		}
	}

	// Assistant reply primer (<|im_start|>assistant<|im_sep|>)
	total += 3
	return total
}
