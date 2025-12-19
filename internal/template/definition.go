package template

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const ComposeFilename = "compose.yaml"

type Template struct {
	Metadata Metadata
	Services []Service
}

type Service struct {
	Name string
	Data map[string]any
}

type Metadata struct {
	Name        string
	Description string
	Features    []string
	Args        []Arg
}

type Arg struct {
	Name        string
	Description string
	Required    bool
	Example     string
}

func FromContent(reader io.Reader) (Template, error) {
	type composeFile struct {
		Services map[string]any `yaml:"services"`
		XTopo    Metadata       `yaml:"x-topo"`
	}

	var parsed composeFile
	decoder := yaml.NewDecoder(reader)
	if err := decoder.Decode(&parsed); err != nil {
		return Template{}, fmt.Errorf("failed to decode template: %w", err)
	}

	var services []Service
	for name, svc := range parsed.Services {
		services = append(services, Service{
			Data: svc.(map[string]any),
			Name: name,
		})
	}

	return Template{
		Services: services,
		Metadata: parsed.XTopo,
	}, nil
}

func FromDir(destDir string) (Template, error) {
	composeServicePath := filepath.Join(destDir, ComposeFilename)

	f, err := os.Open(composeServicePath)
	if err != nil {
		return Template{}, err
	}
	defer func() { _ = f.Close() }()

	return FromContent(f)
}

type rawMetadata struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	Features    []string          `yaml:"features,omitempty"`
	Args        map[string]rawArg `yaml:"args,omitempty"`
}

type rawArg struct {
	Description string `yaml:"description"`
	Required    bool   `yaml:"required"`
	Example     string `yaml:"example,omitempty"`
}

func (t *Metadata) UnmarshalYAML(node *yaml.Node) error {
	var raw rawMetadata
	if err := node.Decode(&raw); err != nil {
		return err
	}

	t.Name = raw.Name
	t.Description = raw.Description
	t.Features = raw.Features
	t.Args = parseArgsInOrder(node, raw.Args)

	return nil
}

func parseArgsInOrder(node *yaml.Node, argsMap map[string]rawArg) []Arg {
	var result []Arg

	for i := 0; i < len(node.Content); i += 2 {
		if node.Content[i].Value == "args" {
			argsNode := node.Content[i+1]
			for j := 0; j < len(argsNode.Content); j += 2 {
				name := argsNode.Content[j].Value
				if metadata, ok := argsMap[name]; ok {
					result = append(result, Arg{
						Name:        name,
						Description: metadata.Description,
						Required:    metadata.Required,
						Example:     metadata.Example,
					})
				}
			}
			break
		}
	}

	return result
}
