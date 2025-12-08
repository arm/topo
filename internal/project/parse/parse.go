package parse

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/arm-debug/topo-cli/internal/arguments"
	"github.com/compose-spec/compose-go/v2/cli"
	"github.com/compose-spec/compose-go/v2/types"
	"gopkg.in/yaml.v3"
)

func ListArgs(root *yaml.Node) []arguments.Arg {
	result := []arguments.Arg{}

	xTopo := find(root, "x-topo")
	if xTopo == nil {
		return result
	}

	args := find(xTopo, "args")
	if args == nil {
		return result
	}

	for i := 0; i < len(args.Content); i += 2 {
		k := args.Content[i]
		v := args.Content[i+1]
		arg := arguments.Arg{
			Name: k.Value,
		}
		if desc := find(v, "description"); desc != nil {
			arg.Description = desc.Value
		}
		if req := find(v, "required"); req != nil {
			if b, err := strconv.ParseBool(req.Value); err == nil {
				arg.Required = b
			}
		}
		if ex := find(v, "example"); ex != nil {
			arg.Example = ex.Value
		}
		result = append(result, arg)
	}

	return result
}

func ApplyArgs(root *yaml.Node, resolved []arguments.ResolvedArg, w io.Writer) error {
	if resolved == nil {
		return nil
	}

	services := find(root, "services")
	if services == nil {
		return nil
	}

	used := make(map[string]bool, len(resolved))

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
			applyArgsMappingNode(args, resolved, used)
		case yaml.SequenceNode:
			applyArgsSequenceNode(args, resolved, used)
		default:
			return fmt.Errorf("unsupported YAML node kind for build.args: %v", args.Kind)
		}
	}

	for _, a := range resolved {
		if !used[a.Name] && w != nil {
			fmt.Fprintf(w, "warning: arg %q was resolved but not found in any service build args\n", a.Name)
		}
	}
	return nil
}

func applyArgsMappingNode(args *yaml.Node, resolved []arguments.ResolvedArg, used map[string]bool) {
	for j := 0; j < len(args.Content); j += 2 {
		k := args.Content[j]
		v := args.Content[j+1]
		for _, a := range resolved {
			if k.Value == a.Name {
				v.Value = a.Value
				used[a.Name] = true
			}
		}
	}
}

func applyArgsSequenceNode(args *yaml.Node, resolved []arguments.ResolvedArg, used map[string]bool) {
	for _, node := range args.Content {
		name := node.Value

		// Extract name from key=value form
		eq := strings.Index(name, "=")
		if eq != -1 {
			name = name[:eq]
		}
		for _, a := range resolved {
			if name == a.Name {
				node.Value = fmt.Sprintf("%s=%s", a.Name, a.Value)
				used[a.Name] = true
			}
		}
	}
}

func Read(targetProjectFile string) (*types.Project, error) {
	ctx := context.Background()
	options, err := cli.NewProjectOptions(
		[]string{targetProjectFile},
		cli.WithOsEnv, cli.WithDotEnv,
		cli.WithResolvedPaths(false),
		cli.WithNormalization(false),
	)
	if err != nil {
		return nil, err
	}
	project, err := cli.ProjectFromOptions(ctx, options)
	if err != nil {
		return nil, err
	}
	return project, nil
}

func ReadNodes(targetProjectFile string) (*yaml.Node, error) {
	fileData, err := os.ReadFile(targetProjectFile)
	if err != nil {
		return nil, err
	}

	var root yaml.Node
	if err := yaml.Unmarshal(fileData, &root); err != nil {
		return nil, err
	}
	if len(root.Content) == 0 {
		return nil, fmt.Errorf("%s is empty", targetProjectFile)
	}
	doc := root.Content[0]
	return doc, nil
}

func find(m *yaml.Node, key string) *yaml.Node {
	for i := 0; i < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			return m.Content[i+1]
		}
	}
	return nil
}
