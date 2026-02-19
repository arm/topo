package compose

import (
	"bytes"
	"fmt"
	"io"
	"sort"
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

func ApplyArgs(root *yaml.Node, toApply map[string]string) ([]logger.Entry, error) {
	if len(toApply) == 0 {
		return []logger.Entry{{
			Level:   logger.Info,
			Message: "no args to apply",
		}}, nil
	}

	services := find(root, "services")
	if services == nil {
		return []logger.Entry{{
			Level:   logger.Info,
			Message: "no services to apply args",
		}}, nil
	}

	used := make(map[string]bool, len(toApply))
	var entries []logger.Entry

	for i := 0; i < len(services.Content); i += 2 {
		svc := services.Content[i+1]
		hasExtends := find(svc, "extends") != nil
		build := find(svc, "build")

		if build == nil {
			if !hasExtends {
				continue
			}
			argsNode := newArgsMappingNode(toApply, used)
			build = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
			build.Content = append(build.Content, scalarNode("args"), argsNode)
			svc.Content = append(svc.Content, scalarNode("build"), build)
			continue
		}

		args := find(build, "args")
		if args == nil {
			if !hasExtends {
				continue
			}
			argsNode := newArgsMappingNode(toApply, used)
			build.Content = append(build.Content, scalarNode("args"), argsNode)
			continue
		}

		switch args.Kind {
		case yaml.MappingNode:
			applyArgsMappingNode(args, toApply, used)
		case yaml.SequenceNode:
			applyArgsSequenceNode(args, toApply, used)
		default:
			return nil, fmt.Errorf("unsupported YAML node kind for build.args: %v", args.Kind)
		}
	}

	for argName := range toApply {
		if !used[argName] {
			entries = append(entries, logger.Entry{
				Level:   logger.Warning,
				Message: fmt.Sprintf("arg %q was resolved but not found in any service build args", argName),
			})
		}
	}
	return entries, nil
}

func RemoveService(project *yaml.Node, serviceName string) error {
	services := find(project, "services")
	if services != nil {
		for i := 0; i < len(services.Content); i += 2 {
			if services.Content[i].Value == serviceName {
				services.Content = append(services.Content[:i], services.Content[i+2:]...)
				return nil
			}
		}
	}
	return fmt.Errorf("service %s not found", serviceName)
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

func scalarNode(value string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Value: value, Tag: "!!str"}
}

func newArgsMappingNode(toApply map[string]string, used map[string]bool) *yaml.Node {
	node := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	keys := make([]string, 0, len(toApply))
	for k := range toApply {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, name := range keys {
		node.Content = append(node.Content, scalarNode(name), scalarNode(toApply[name]))
		used[name] = true
	}
	return node
}

func find(m *yaml.Node, key string) *yaml.Node {
	for i := 0; i < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			return m.Content[i+1]
		}
	}
	return nil
}
