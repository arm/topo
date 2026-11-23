// Package suggestions defines configuration candidates and explicitly selected actions.
package suggestions

import (
	"context"

	"github.com/arm/topo/pkg/plugin/observations"
)

type Provider interface {
	Suggestions(context.Context, Request) (Response, error)
	// RunAction must stop and release resources when its context is cancelled.
	// Interactive hosts cancel actions when the user submits a value or leaves the prompt.
	RunAction(context.Context, ActionID, Request) (ActionResponse, error)
}

// Request concerns one parameter. CurrentValue includes staged changes;
// nil means absent, while a pointer to "" means present but empty.
type Request struct {
	Project            ProjectMetadata
	Hints              Hints
	CurrentValue       *string
	TargetObservations observations.Tree
}

// ProjectMetadata describes the project being configured, using x-topo metadata.
type ProjectMetadata struct {
	// Name is empty when x-topo.name is absent.
	Name string
}

type Response struct {
	Suggestions      []Suggestion
	AvailableActions []Action
}

type ActionID string

type Action struct {
	ID          ActionID
	Label       string
	Description string
}

type Suggestion struct {
	Value string
	Label string
}

type ActionResponse struct {
	Suggestions []Suggestion
}
