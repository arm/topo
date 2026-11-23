package main

import (
	"strings"

	"github.com/arm/topo/pkg/plugin/suggestions"
)

// BuildModelQuery interprets advisory project hints and target observations as Portal filters.
// Unsupported or malformed metadata leaves the corresponding filter unset.
func BuildModelQuery(request suggestions.Request) ModelQuery {
	query := ModelQuery{ProjectName: strings.TrimSpace(request.Project.Name)}
	for key, destination := range map[string]*string{
		"model.task": &query.Task, "model.format": &query.Format,
	} {
		var value string
		if err := request.Hints.Decode(key, &value); err == nil {
			*destination = value
		}
	}
	switch query.Format {
	case "onnx":
		query.Runtime = "onnx"
	case "pte":
		query.Runtime = "executorch"
		var backend string
		if err := request.Hints.Decode("executorch.backend", &backend); err == nil {
			backend = strings.ToLower(strings.TrimSpace(backend))
			// Portal distinguishes execution backends, not ExecuTorch delegates.
			if backend == "ethos-u" {
				query.Backend = "ethos-u"
			} else if backend != "" {
				query.Backend = "cpu"
			}
		}
	}

	var memoryBytes int64
	if err := request.TargetObservations.Decode("example.memory.target.memory.total_bytes", &memoryBytes); err == nil {
		// Portal uses decimal MB. Total target RAM is only a discovery ceiling, not available workload memory.
		if memoryBytes > 0 {
			query.PeakMemoryMB = int(memoryBytes / 1_000_000)
		}
	}
	return query
}
