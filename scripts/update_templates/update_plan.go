package main

import (
	"fmt"
	"strings"
)

type UpdatePlan struct {
	ToAdd     []GitHubSource
	ToUpdate  []GitHubSource
	ToRemove  []Template
	Unchanged []Template
}

func (p UpdatePlan) HasChanges() bool {
	return len(p.ToAdd) > 0 || len(p.ToUpdate) > 0 || len(p.ToRemove) > 0
}

func (p UpdatePlan) String() string {
	var lines []string
	lines = append(lines, fmt.Sprintf("🆕 %d to add", len(p.ToAdd)))
	lines = appendSourceURLs(lines, p.ToAdd)
	lines = append(lines, fmt.Sprintf("🔄 %d to update", len(p.ToUpdate)))
	lines = appendSourceURLs(lines, p.ToUpdate)
	lines = append(lines, fmt.Sprintf("🗑️ %d to remove", len(p.ToRemove)))
	lines = appendTemplateURLs(lines, p.ToRemove)
	lines = append(lines, fmt.Sprintf("☑️ %d unchanged", len(p.Unchanged)))
	return strings.Join(lines, "\n")
}

func appendSourceURLs(lines []string, sources []GitHubSource) []string {
	for _, source := range sources {
		lines = append(lines, fmt.Sprintf("  - %s", source.URL()))
	}
	return lines
}

func appendTemplateURLs(lines []string, templates []Template) []string {
	for _, template := range templates {
		lines = append(lines, fmt.Sprintf("  - %s", template.URL))
	}
	return lines
}

func PlanUpdate(sources []GitHubSource, current []Template) UpdatePlan {
	sourceByID := make(map[TemplateSourceID]GitHubSource, len(sources))
	for _, source := range sources {
		sourceByID[source.ID()] = source
	}

	currentByID := make(map[TemplateSourceID]Template, len(current))
	for _, template := range current {
		currentByID[template.SourceID()] = template
	}

	var plan UpdatePlan
	for _, source := range sources {
		template, exists := currentByID[source.ID()]
		if !exists {
			plan.ToAdd = append(plan.ToAdd, source)
			continue
		}

		if template.Ref != source.SHA {
			plan.ToUpdate = append(plan.ToUpdate, source)
			continue
		}

		plan.Unchanged = append(plan.Unchanged, template)
	}

	for _, template := range current {
		if _, exists := sourceByID[template.SourceID()]; !exists {
			plan.ToRemove = append(plan.ToRemove, template)
		}
	}

	return plan
}
