package main

import "fmt"

func ProjectsInSourceOrder(sources []GitHubSource, projects []Project) []Project {
	projectByID := make(map[ProjectSourceID]Project, len(projects))
	for _, project := range projects {
		projectByID[project.SourceID()] = project
	}

	ordered := make([]Project, 0, len(sources))
	for _, source := range sources {
		project, exists := projectByID[source.ID()]
		if !exists {
			panic(fmt.Sprintf("missing project for source %s", source))
		}
		ordered = append(ordered, project)
	}

	return ordered
}
