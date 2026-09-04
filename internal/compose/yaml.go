package compose

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/arm/topo/internal/output/logger"
	"gopkg.in/yaml.v3"
)

func ReadNode(composeFile io.Reader) (*yaml.Node, error) {
	fileData, err := io.ReadAll(composeFile)
	if err != nil {
		return nil, err
	}

	var root yaml.Node
	if err := yaml.Unmarshal(fileData, &root); err != nil {
		return nil, err
	}
	if len(root.Content) == 0 {
		return nil, fmt.Errorf("compose file is empty")
	}
	doc := root.Content[0]
	return doc, nil
}

func ApplyParameters(root *yaml.Node, parameters map[string]string) error {
	if len(parameters) == 0 {
		logger.Info("no parameters to apply")
		return nil
	}

	services := find(root, "services")
	if services == nil {
		logger.Info("no services to apply parameters to")
		return nil
	}

	used := make(map[string]bool, len(parameters))

	for i := 0; i < len(services.Content); i += 2 {
		svc := services.Content[i+1]
		build := find(svc, "build")
		args := find(build, "args")

		if args == nil {
			if find(svc, "extends") != nil {
				name := services.Content[i].Value
				logger.Warn(fmt.Sprintf("service %q uses `extends` without `build.args`; inherited build args will not be configured. declare the required args on this service, or set `build.args: {}` to silence this warning", name))
			}

			continue
		}

		switch args.Kind {
		case yaml.MappingNode:
			applyArgsMappingNode(args, parameters, used)
		case yaml.SequenceNode:
			applyArgsSequenceNode(args, parameters, used)
		default:
			return fmt.Errorf("unsupported YAML node kind for build.args: %v", args.Kind)
		}
	}

	for name := range parameters {
		if !used[name] {
			logger.Warn(fmt.Sprintf("parameter %q was provided but not found in any service build args", name))
		}
	}
	return nil
}

func WriteNode(project *yaml.Node, target io.Writer) error {
	buf := &bytes.Buffer{}
	enc := yaml.NewEncoder(buf)
	enc.SetIndent(2)
	if err := enc.Encode(project); err != nil {
		return err
	}
	_ = enc.Close()
	if _, err := target.Write(buf.Bytes()); err != nil {
		return err
	}
	return nil
}

func applyArgsMappingNode(args *yaml.Node, parameters map[string]string, used map[string]bool) {
	for j := 0; j < len(args.Content); j += 2 {
		key := args.Content[j]
		value := args.Content[j+1]
		for param, provided := range parameters {
			if key.Value == param {
				value.Value = provided
				used[param] = true
			}
		}
	}
}

func applyArgsSequenceNode(args *yaml.Node, parameters map[string]string, used map[string]bool) {
	for _, node := range args.Content {
		name := node.Value

		// Extract name from key=value form
		eq := strings.Index(name, "=")
		if eq != -1 {
			name = name[:eq]
		}
		for param, provided := range parameters {
			if name == param {
				node.Value = fmt.Sprintf("%s=%s", param, provided)
				used[param] = true
			}
		}
	}
}

func find(m *yaml.Node, key string) *yaml.Node {
	if m == nil {
		return nil
	}

	for i := 0; i < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			return m.Content[i+1]
		}
	}
	return nil
}
