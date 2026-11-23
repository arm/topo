package parameter

import (
	"context"
	"fmt"

	"github.com/arm/topo/internal/output/term"
	"github.com/arm/topo/pkg/plugin/suggestions"
)

func (r *InteractiveResolver) parameterActions(parameter Parameter) ([]term.PromptAction, string, error) {
	ctx := r.Context
	if ctx == nil {
		ctx = context.Background()
	}
	request := suggestions.Request{
		Project:            parameter.Project,
		Hints:              parameter.Hints,
		CurrentValue:       parameter.ExistingValue,
		TargetObservations: r.TargetObservations,
	}
	// These control keys do not replace navigation, editing, submission, or cancellation.
	keys := []rune{15, 7, 24, 29, 30, 31, 28}
	var actions []term.PromptAction
	labels := ""
	for _, provider := range r.Providers {
		response, err := provider.Suggestions(ctx, request)
		if err != nil {
			return nil, "", err
		}
		for _, action := range response.AvailableActions {
			if len(actions) == len(keys) {
				return nil, "", fmt.Errorf("too many actions for parameter %s", parameter.Name)
			}
			key := keys[len(actions)]
			labels += r.palette.Color(term.Dim, fmt.Sprintf(" · Ctrl+%c to %s", key+'@', action.Label))
			actions = append(actions, term.PromptAction{Key: key, Run: func(ctx context.Context, input string) (string, error) {
				fresh := request
				if input != "" || parameter.ExistingValue != nil {
					fresh.CurrentValue = &input
				}
				result, err := provider.RunAction(ctx, action.ID, fresh)
				if err != nil {
					return "", err
				}
				if len(result.Suggestions) == 0 {
					return input, nil
				}
				return result.Suggestions[0].Value, nil
			}})
		}
	}
	return actions, labels, nil
}
