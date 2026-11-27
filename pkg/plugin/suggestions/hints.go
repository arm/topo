package suggestions

import (
	"encoding/json"
	"fmt"
)

type Hints map[string]json.RawMessage

func (h Hints) Decode(key string, destination any) error {
	value, ok := h[key]
	if !ok {
		return fmt.Errorf("hint %q unavailable", key)
	}
	if err := json.Unmarshal(value, destination); err != nil {
		return fmt.Errorf("decoding hint %q: %w", key, err)
	}
	return nil
}
