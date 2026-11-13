package models

import (
	"encoding/json"
	"reflect"
	"strings"
)

// chatRequestAlias has the fields of ChatCompletionRequest without its JSON methods.
type chatRequestAlias ChatCompletionRequest

var modeledRequestFields = func() map[string]bool {
	names := map[string]bool{}
	t := reflect.TypeOf(ChatCompletionRequest{})
	for i := 0; i < t.NumField(); i++ {
		name, _, _ := strings.Cut(t.Field(i).Tag.Get("json"), ",")
		if name != "" && name != "-" {
			names[name] = true
		}
	}
	return names
}()

// UnmarshalJSON decodes the declared fields and keeps every other field in Extra.
func (r *ChatCompletionRequest) UnmarshalJSON(data []byte) error {
	var alias chatRequestAlias
	if err := json.Unmarshal(data, &alias); err != nil {
		return err
	}
	var all map[string]json.RawMessage
	if err := json.Unmarshal(data, &all); err != nil {
		return err
	}
	*r = ChatCompletionRequest(alias)
	for name, raw := range all {
		// encoding/json matches declared fields case-insensitively, so compare the same way.
		if modeledRequestFields[strings.ToLower(name)] {
			continue
		}
		if r.Extra == nil {
			r.Extra = map[string]json.RawMessage{}
		}
		r.Extra[name] = raw
	}
	return nil
}

// MarshalJSON encodes the declared fields and adds back the fields held in Extra.
func (r ChatCompletionRequest) MarshalJSON() ([]byte, error) {
	base, err := json.Marshal(chatRequestAlias(r))
	if err != nil || len(r.Extra) == 0 {
		return base, err
	}
	var merged map[string]json.RawMessage
	if err := json.Unmarshal(base, &merged); err != nil {
		return nil, err
	}
	for name, raw := range r.Extra {
		if _, declared := merged[name]; !declared {
			merged[name] = raw
		}
	}
	return json.Marshal(merged)
}
