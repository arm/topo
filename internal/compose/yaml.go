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

func ApplyArgs(root *yaml.Node, toApply map[string]string) error {
	if len(toApply) == 0 {
		logger.Info("no args to apply")
		return nil
	}

	services := find(root, "services")
	if services == nil {
		logger.Info("no services to apply args")
		return nil
	}

	used := make(map[string]bool, len(toApply))

	for i := 0; i < len(services.Content); i += 2 {
		svc := services.Content[i+1]
		addMissing := find(svc, "extends") != nil

		build := find(svc, "build")
		if build == nil {
			if !addMissing {
				continue
			}
			build = mappingNode()
			svc.Content = append(svc.Content, scalarNode("build"), build)
		}

		args := find(build, "args")
		if args == nil {
			if !addMissing {
				continue
			}
			args = mappingNode()
			build.Content = append(build.Content, scalarNode("args"), args)
		}

		switch args.Kind {
		case yaml.MappingNode:
			applyArgsMappingNode(args, toApply, used, addMissing)
		case yaml.SequenceNode:
			applyArgsSequenceNode(args, toApply, used, addMissing)
		default:
			return fmt.Errorf("unsupported YAML node kind for build.args: %v", args.Kind)
		}
	}

	for argName := range toApply {
		if !used[argName] {
			logger.Warn(fmt.Sprintf("arg %q was resolved but not found in any service build args", argName))
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

func applyArgsMappingNode(args *yaml.Node, toApply map[string]string, used map[string]bool, add bool) {
	changed := map[string]bool{}

	for j := 0; j < len(args.Content); j += 2 {
		k := args.Content[j]
		v := args.Content[j+1]
		for argName, argValue := range toApply {
			if k.Value == argName {
				v.Value = argValue
				used[argName] = true
				changed[argName] = true
			}
		}
	}

	if add && len(changed) < len(toApply) {
		for argName, argValue := range toApply {
			if !changed[argName] {
				args.Content = append(args.Content, scalarNode(argName), scalarNode(argValue))
				used[argName] = true
			}
		}
	}
}

func applyArgsSequenceNode(args *yaml.Node, toApply map[string]string, used map[string]bool, add bool) {
	changed := map[string]bool{}

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
				changed[argName] = true
			}
		}
	}

	if add && len(changed) < len(toApply) {
		for argName, argValue := range toApply {
			if !changed[argName] {
				args.Content = append(args.Content, scalarNode(fmt.Sprintf("%s=%s", argName, argValue)))
				used[argName] = true
			}
		}
	}
}

func scalarNode(value string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Value: value, Tag: "!!str"}
}

func mappingNode() *yaml.Node {
	return &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
}

func find(m *yaml.Node, key string) *yaml.Node {
	for i := 0; i < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			return m.Content[i+1]
		}
	}
	return nil
}
