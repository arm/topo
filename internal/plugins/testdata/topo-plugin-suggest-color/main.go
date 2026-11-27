package main

import (
	"context"
	"fmt"

	"github.com/arm/topo/pkg/plugin"
	"github.com/arm/topo/pkg/plugin/suggestions"
)

func main() {
	plugin.ServeSuggestions("test.suggest-color", provider{})
}

type provider struct{}

func (provider) Suggestions(context.Context, suggestions.Request) (suggestions.Response, error) {
	return suggestions.Response{
		Suggestions: []suggestions.Suggestion{
			{Value: "blue", Label: "Blue"},
			{Value: "green", Label: "Green"},
		},
		AvailableActions: []suggestions.Action{{ID: "use-green", Label: "Use green"}},
	}, nil
}

func (provider) RunAction(
	_ context.Context, id suggestions.ActionID, _ suggestions.Request,
) (suggestions.ActionResponse, error) {
	if id != "use-green" {
		return suggestions.ActionResponse{}, fmt.Errorf("unknown action %q", id)
	}
	return suggestions.ActionResponse{
		Suggestions: []suggestions.Suggestion{{Value: "green", Label: "Green"}},
	}, nil
}
