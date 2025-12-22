package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/arm-debug/topo-cli/internal/template"
)

func BuildTemplate(repoURL string, compose io.Reader) (Template, error) {
	tmpl, err := template.FromContent(compose)
	if err != nil {
		return Template{}, fmt.Errorf("parse compose definition: %w", err)
	}

	metadata := tmpl.Metadata
	if metadata.Name == "" {
		return Template{}, fmt.Errorf("no valid x-topo name in compose definition")
	}

	return Template{
		ID:          makeTemplateID(metadata.Name),
		Description: metadata.Description,
		Features:    metadata.Features,
		URL:         repoURL,
	}, nil
}

func parseRepoSpec(spec string) (repo, ref string) {
	parts := strings.SplitN(spec, "#", 2)
	repo = parts[0]
	if len(parts) == 2 {
		ref = parts[1]
	}
	return
}

func makeTemplateID(name string) string {
	parts := strings.Fields(name)
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, "-")
}
