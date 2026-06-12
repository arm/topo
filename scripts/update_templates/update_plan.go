package main

type UpdatePlan struct {
	ToAdd     []GitHubSource
	ToUpdate  []GitHubSource
	ToRemove  []Template
	Unchanged []Template
}

// TODO: codify the fact CloneURL and URL are the foreign key id

func PlanUpdate(sources []GitHubSource, current []Template) UpdatePlan {
	currentByURL := make(map[string]Template, len(current))
	for _, template := range current {
		currentByURL[template.URL] = template
	}

	sourceByURL := make(map[string]GitHubSource, len(sources))
	for _, source := range sources {
		url := source.CloneURL()
		sourceByURL[url] = source
	}

	var plan UpdatePlan
	for _, source := range sources {
		url := source.CloneURL()
		template, exists := currentByURL[url]
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
		if _, exists := sourceByURL[template.URL]; !exists {
			plan.ToRemove = append(plan.ToRemove, template)
		}
	}

	return plan
}
