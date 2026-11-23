package observations

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

type Tree map[string]any

// UnmarshalJSON preserves integer precision when reading observation trees.
func (t *Tree) UnmarshalJSON(data []byte) error {
	var values map[string]any
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(&values); err != nil {
		return err
	}
	*t = Tree(values)
	return nil
}

func (t Tree) Decode(path string, destination any) error {
	data, err := json.Marshal(t)
	if err != nil {
		return fmt.Errorf("encoding observations: %w", err)
	}
	for segment := range strings.SplitSeq(path, ".") {
		var properties map[string]json.RawMessage
		if err := json.Unmarshal(data, &properties); err != nil {
			return fmt.Errorf("decoding observation %q: %w", path, err)
		}
		value, ok := properties[segment]
		if !ok {
			return fmt.Errorf("observation %q unavailable", path)
		}
		data = value
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(destination); err != nil {
		return fmt.Errorf("decoding observation %q: %w", path, err)
	}
	return nil
}
