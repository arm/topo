package main

import (
	"context"
	"fmt"
	"io"

	"github.com/arm/topo/pkg/plugin/observations"
)

func observePluginTargets(
	ctx context.Context, failures io.Writer, plugins []observations.Provider, runner observations.TargetRunner,
) (observations.Tree, error) {
	reports := make(observations.Tree)
	for _, plugin := range plugins {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		result, err := plugin.ObserveTarget(ctx, runner)
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if err == nil {
			if err := mergeObservationTrees(reports, result); err != nil {
				return nil, fmt.Errorf("merging observations: %w", err)
			}
			continue
		}
		if err := reportPluginFailures(failures, []error{err}); err != nil {
			return nil, err
		}
	}
	return reports, ctx.Err()
}

func mergeObservationTrees(destination, source observations.Tree) error {
	type branch struct {
		destination observations.Tree
		source      observations.Tree
		path        string
	}
	pending := []branch{{destination: destination, source: source}}
	for len(pending) > 0 {
		current := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		for key, value := range current.source {
			existing, found := current.destination[key]
			if !found {
				current.destination[key] = value
				continue
			}
			left, leftOK := observationObject(existing)
			right, rightOK := observationObject(value)
			path := current.path + key
			if !leftOK || !rightOK {
				return fmt.Errorf("conflicting observation %q", path)
			}
			pending = append(pending, branch{destination: left, source: right, path: path + "."})
		}
	}
	return nil
}

func observationObject(value any) (observations.Tree, bool) {
	switch value := value.(type) {
	case observations.Tree:
		return value, value != nil
	case map[string]any:
		return observations.Tree(value), value != nil
	default:
		return nil, false
	}
}

func reportPluginFailures(writer io.Writer, failures []error) error {
	for _, failure := range failures {
		if _, err := fmt.Fprintf(writer, "Plugin failure: %v\n", failure); err != nil {
			return fmt.Errorf("reporting plugin failure: %w", err)
		}
	}
	return nil
}
