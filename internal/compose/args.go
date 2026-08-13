package compose

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/arm/topo/internal/output/logger"
	"github.com/compose-spec/compose-go/v2/types"
	"gopkg.in/yaml.v3"
)

func ApplyArgs(composeFilePath string, toApply map[string]string) error {
	if len(toApply) == 0 {
		logger.Info("no args to apply")
		return nil
	}

	project, err := ReadProject(composeFilePath)
	if err != nil {
		return fmt.Errorf("failed to read project from %s: %w", composeFilePath, err)
	}

	root, err := readNode(composeFilePath)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", composeFilePath, err)
	}

	services := find(root, "services")
	if services == nil {
		logger.Info("no services to apply args")
		return nil
	}

	used := make(map[string]bool, len(toApply))

	for i := 0; i < len(services.Content); i += 2 {
		name := services.Content[i].Value
		svc := services.Content[i+1]

		fullSvc, ok := project.Services[name]
		if !ok {
			return fmt.Errorf("service %s not found in fully-qualified project", name)
		}

		err = applyArgs(svc, fullSvc, toApply, used)
		if err != nil {
			return fmt.Errorf("failed to apply args to service %s: %w", name, err)
		}
	}

	for argName := range toApply {
		if !used[argName] {
			logger.Warn(fmt.Sprintf("arg %q was resolved but not found in any service build args", argName))
		}
	}

	err = writeNode(composeFilePath, root)
	if err != nil {
		return fmt.Errorf("failed to write updated compose file: %w", err)
	}

	return nil
}

func writeNode(composeFilePath string, project *yaml.Node) error {
	buf := &bytes.Buffer{}
	enc := yaml.NewEncoder(buf)
	enc.SetIndent(2)
	if err := enc.Encode(project); err != nil {
		return err
	}
	_ = enc.Close()
	if err := os.WriteFile(composeFilePath, buf.Bytes(), 0644); err != nil {
		return err
	}
	return nil
}

func readNode(composeFilePath string) (*yaml.Node, error) {
	fileData, err := os.ReadFile(composeFilePath)
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

func applyArgs(svc *yaml.Node, fullSvc types.ServiceConfig, toApply map[string]string, used map[string]bool) error {
	build := find(svc, "build")
	hasBuild := build != nil
	if build == nil {
		build = mappingNode()
	}

	args := find(build, "args")
	hasArgs := args != nil
	if args == nil {
		args = mappingNode()
	}

	fullArgs := types.MappingWithEquals{}
	if fullSvc.Build != nil {
		fullArgs = fullSvc.Build.Args
	}

	appliedAny := false
	for argName, argValue := range toApply {
		_, shouldAppend := fullArgs[argName]
		applied, err := applyArg(args, argName, argValue, shouldAppend)
		if err != nil {
			return fmt.Errorf("failed to apply arg %q: %w", argName, err)
		}
		if applied {
			used[argName] = true
			appliedAny = true
		}
	}

	if appliedAny {
		if !hasArgs {
			build.Content = append(build.Content, scalarNode("args"), args)
		}
		if !hasBuild {
			svc.Content = append(svc.Content, scalarNode("build"), build)
		}
	}

	return nil
}

func applyArg(args *yaml.Node, argName string, argValue string, shouldAppend bool) (bool, error) {
	switch args.Kind {
	case yaml.MappingNode:
		return applyArgToMappingNode(args, argName, argValue, shouldAppend), nil
	case yaml.SequenceNode:
		return applyArgSequenceNode(args, argName, argValue, shouldAppend), nil
	default:
		return false, fmt.Errorf("unsupported YAML node kind for build.args: %v", args.Kind)
	}
}

func applyArgToMappingNode(args *yaml.Node, argName string, argValue string, shouldAppend bool) bool {
	for j := 0; j < len(args.Content); j += 2 {
		k := args.Content[j]
		v := args.Content[j+1]
		if k.Value == argName {
			v.Value = argValue
			return true
		}
	}

	if shouldAppend {
		args.Content = append(args.Content, scalarNode(argName), scalarNode(argValue))
		return true
	}

	return false
}

func applyArgSequenceNode(args *yaml.Node, argName string, argValue string, shouldAppend bool) bool {
	for _, node := range args.Content {
		name := node.Value

		// Extract name from key=value form
		eq := strings.Index(name, "=")
		if eq != -1 {
			name = name[:eq]
		}

		if name == argName {
			node.Value = fmt.Sprintf("%s=%s", argName, argValue)
			return true
		}
	}

	if shouldAppend {
		args.Content = append(args.Content, scalarNode(fmt.Sprintf("%s=%s", argName, argValue)))
		return true
	}

	return false
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
