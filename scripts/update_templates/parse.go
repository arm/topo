package main

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/arm/topo/internal/template"
	"gopkg.in/yaml.v3"
)

func BuildTemplate(repoURL string, compose io.Reader) (Template, error) {
	content, err := io.ReadAll(compose)
	if err != nil {
		return Template{}, fmt.Errorf("failed to read compose definition: %w", err)
	}

	tmpl, err := template.FromContent(bytes.NewReader(content))
	if err != nil {
		return Template{}, fmt.Errorf("failed to parse compose definition: %w", err)
	}

	metadata := tmpl.Metadata
	if metadata.Name == "" {
		return Template{}, fmt.Errorf("no valid x-topo name in compose definition")
	}

	args, err := parseArgs(content)
	if err != nil {
		return Template{}, fmt.Errorf("failed to parse args: %w", err)
	}

	return Template{
		Name:        metadata.Name,
		Description: metadata.Description,
		Features:    metadata.Features,
		Args:        args,
		URL:         repoURL,
	}, nil
}

func parseArgs(compose []byte) (map[string]Arg, error) {
	type rawArg struct {
		Description string         `yaml:"description"`
		Required    bool           `yaml:"required"`
		Default     string         `yaml:"default"`
		Example     string         `yaml:"example"`
		Hints       map[string]any `yaml:"hints"`
	}
	type rawCompose struct {
		XTopo struct {
			Args map[string]rawArg `yaml:"args"`
		} `yaml:"x-topo"`
	}

	var parsed rawCompose
	if err := yaml.Unmarshal(compose, &parsed); err != nil {
		return nil, err
	}
	if len(parsed.XTopo.Args) == 0 {
		return nil, nil
	}

	args := make(map[string]Arg, len(parsed.XTopo.Args))
	for name, arg := range parsed.XTopo.Args {
		args[name] = Arg{
			Description: arg.Description,
			Required:    arg.Required,
			Default:     arg.Default,
			Example:     arg.Example,
			Hints:       arg.Hints,
		}
	}
	return args, nil
}

func parseRepoSpec(spec string) (repo, ref string) {
	parts := strings.SplitN(spec, "#", 2)
	repo = parts[0]
	if len(parts) == 2 {
		ref = parts[1]
	}
	return
}
