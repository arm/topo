package main

import (
	"fmt"
	"strings"
)

type UpdatePlan struct {
	ToAdd     []GitHubSource
	ToUpdate  []GitHubSource
	ToRemove  []Project
	Unchanged []Project
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
	lines = appendProjectURLs(lines, p.ToRemove)
	lines = append(lines, fmt.Sprintf("☑️ %d unchanged", len(p.Unchanged)))
	return strings.Join(lines, "\n")
}

func appendSourceURLs(lines []string, sources []GitHubSource) []string {
	for _, source := range sources {
		lines = append(lines, fmt.Sprintf("  - %s", source.URL()))
	}
	return lines
}

func appendProjectURLs(lines []string, projects []Project) []string {
	for _, project := range projects {
		lines = append(lines, fmt.Sprintf("  - %s", project.URL))
	}
	return lines
}

func PlanUpdate(sources []GitHubSource, current []Project) UpdatePlan {
	sourceByID := make(map[ProjectSourceID]GitHubSource, len(sources))
	for _, source := range sources {
		sourceByID[source.ID()] = source
	}

	currentByID := make(map[ProjectSourceID]Project, len(current))
	for _, project := range current {
		currentByID[project.SourceID()] = project
	}

	var plan UpdatePlan
	for _, source := range sources {
		project, exists := currentByID[source.ID()]
		if !exists {
			plan.ToAdd = append(plan.ToAdd, source)
			continue
		}

		if project.Ref != source.SHA {
			plan.ToUpdate = append(plan.ToUpdate, source)
			continue
		}

		plan.Unchanged = append(plan.Unchanged, project)
	}

	for _, project := range current {
		if _, exists := sourceByID[project.SourceID()]; !exists {
			plan.ToRemove = append(plan.ToRemove, project)
		}
	}

	return plan
}
