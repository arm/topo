package main

import "fmt"

func TemplatesInSourceOrder(sources []GitHubSource, templates []Template) []Template {
	templateByID := make(map[TemplateSourceID]Template, len(templates))
	for _, template := range templates {
		templateByID[template.SourceID()] = template
	}

	ordered := make([]Template, 0, len(sources))
	for _, source := range sources {
		template, exists := templateByID[source.ID()]
		if !exists {
			panic(fmt.Sprintf("missing template for source %s", source))
		}
		ordered = append(ordered, template)
	}

	return ordered
}
