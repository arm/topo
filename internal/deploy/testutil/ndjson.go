package testutil

import (
	"bytes"
	"encoding/json"
)

type JsonObject map[string]any

func UnmarshalNDJSON(ndJSON []byte) ([]JsonObject, error) {
	objects := []JsonObject{}
	lines := bytes.SplitSeq(bytes.TrimSpace(ndJSON), []byte("\n"))

	for line := range lines {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		var object JsonObject
		if err := json.Unmarshal(line, &object); err != nil {
			return objects, err
		}
		objects = append(objects, object)
	}

	return objects, nil
}
