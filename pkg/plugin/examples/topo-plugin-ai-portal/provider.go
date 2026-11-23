package main

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/arm/topo/pkg/plugin/suggestions"
)

const chooseModel suggestions.ActionID = "choose_model"

type Provider struct {
	PortalURL string
}

func (Provider) Suggestions(ctx context.Context, request suggestions.Request) (suggestions.Response, error) {
	if err := ctx.Err(); err != nil {
		return suggestions.Response{}, err
	}
	if !hasModelHints(request.Hints) {
		return suggestions.Response{}, nil
	}
	return suggestions.Response{
		AvailableActions: []suggestions.Action{{
			ID:          chooseModel,
			Label:       "Choose a model in AI Portal",
			Description: "Open a browser filtered by project hints and observed target memory.",
		}},
	}, nil
}

func hasModelHints(hints suggestions.Hints) bool {
	for _, key := range []string{"model.task", "model.format"} {
		var value string
		if err := hints.Decode(key, &value); err == nil && strings.TrimSpace(value) != "" {
			return true
		}
	}
	var value string
	if err := hints.Decode("meta.type", &value); err == nil && value == "huggingface.repo-id" {
		return true
	}
	var types []string
	return hints.Decode("meta.type", &types) == nil && slices.Contains(types, "huggingface.repo-id")
}

func (p Provider) RunAction(
	ctx context.Context, id suggestions.ActionID, request suggestions.Request,
) (response suggestions.ActionResponse, err error) {
	if err := ctx.Err(); err != nil {
		return suggestions.ActionResponse{}, err
	}
	if id != chooseModel {
		return suggestions.ActionResponse{}, fmt.Errorf("unknown action %q", id)
	}
	// Also bound abandoned browser handoffs while the configuration prompt remains open.
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	query := BuildModelQuery(request)
	callbackServer, err := NewCallbackServer()
	if err != nil {
		return suggestions.ActionResponse{}, err
	}
	defer func() {
		if closeErr := callbackServer.Close(); closeErr != nil && err == nil {
			response, err = suggestions.ActionResponse{}, closeErr
		}
	}()
	portalClient := NewBrowserClient(p.PortalURL)
	if err := portalClient.OpenModelSelectionPage(ctx, callbackServer.CallbackURL(), query); err != nil {
		return suggestions.ActionResponse{}, err
	}
	selection, err := callbackServer.WaitForCallback(ctx)
	if err != nil {
		return suggestions.ActionResponse{}, fmt.Errorf("Portal handoff ended: %w", err)
	}
	return suggestions.ActionResponse{
		Suggestions: []suggestions.Suggestion{{Value: selection.RepoID, Label: selection.RepoID}},
	}, nil
}
