package compose

import (
	"fmt"
	"io"
	"strings"

	"gopkg.in/yaml.v3"
)

func ReadNodes(composeFile io.Reader) (*yaml.Node, error) {
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

func ApplyArgs(root *yaml.Node, toApply map[string]string, w io.Writer) error {
	if len(toApply) == 0 {
		return nil
	}

	services := find(root, "services")
	if services == nil {
		return nil
	}

	used := make(map[string]bool, len(toApply))

	for i := 0; i < len(services.Content); i += 2 {
		svc := services.Content[i+1]
		build := find(svc, "build")
		if build == nil {
			continue
		}
		args := find(build, "args")
		if args == nil {
			continue
		}

		switch args.Kind {
		case yaml.MappingNode:
			applyArgsMappingNode(args, toApply, used)
		case yaml.SequenceNode:
			applyArgsSequenceNode(args, toApply, used)
		default:
			return fmt.Errorf("unsupported YAML node kind for build.args: %v", args.Kind)
		}
	}

	for argName := range toApply {
		if !used[argName] && w != nil {
			_, err := fmt.Fprintf(w, "warning: arg %q was resolved but not found in any service build args\n", argName)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func applyArgsMappingNode(args *yaml.Node, toApply map[string]string, used map[string]bool) {
	for j := 0; j < len(args.Content); j += 2 {
		k := args.Content[j]
		v := args.Content[j+1]
		for argName, argValue := range toApply {
			if k.Value == argName {
				v.Value = argValue
				used[argName] = true
			}
		}
	}
}

func applyArgsSequenceNode(args *yaml.Node, toApply map[string]string, used map[string]bool) {
	for _, node := range args.Content {
		name := node.Value

		// Extract name from key=value form
		eq := strings.Index(name, "=")
		if eq != -1 {
			name = name[:eq]
		}
		for argName, argValue := range toApply {
			if name == argName {
				node.Value = fmt.Sprintf("%s=%s", argName, argValue)
				used[argName] = true
			}
		}
	}
}

func find(m *yaml.Node, key string) *yaml.Node {
	for i := 0; i < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			return m.Content[i+1]
		}
	}
	return nil
}
