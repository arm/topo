package parameter

import (
	"context"
	"fmt"

	"github.com/arm/topo/internal/fact"
	"github.com/arm/topo/internal/runner"
)

type FactsResolver struct {
	providers map[string]fact.Provider
	ctx       context.Context
	runner    runner.CommandRunner
}

func NewFactsResolver(ctx context.Context, targetRunner runner.CommandRunner, providers map[string]fact.Provider) *FactsResolver {
	return &FactsResolver{ctx: ctx, runner: targetRunner, providers: providers}
}

func (r *FactsResolver) Resolve(parameters []Parameter) (Changes, error) {
	sourceable, err := SourceableParameters(parameters, r.providers)
	if err != nil {
		return nil, err
	}
	changes := Changes{}
	results := map[string]fact.Report{}
	for _, parameter := range sourceable {
		source := parameter.Source
		result, ok := results[source.Provider]
		if !ok {
			var err error
			result, err = r.providers[source.Provider].Facts(r.ctx, r.runner)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve %s: %w", parameter.Name, err)
			}
			if err := result.Validate(); err != nil {
				return nil, fmt.Errorf("failed to resolve %s: %w", parameter.Name, err)
			}
			results[source.Provider] = result
		}
		fact, ok := result.Find(source.Group, source.Fact)
		if !ok {
			return nil, fmt.Errorf("fact %q not found in group %q for %s", source.Fact, source.Group, parameter.Name)
		}
		changes[parameter.Name] = new(fact.Value.String())
	}
	return changes, nil
}

func SourceableParameters(parameters []Parameter, providers map[string]fact.Provider) ([]Parameter, error) {
	var sourceable []Parameter
	for _, param := range parameters {
		source := param.Source
		if source == nil {
			continue
		}
		if source.Provider == "" || source.Group == "" || source.Fact == "" {
			return nil, fmt.Errorf("source requires provider, group and fact for %s", param.Name)
		}
		if _, exists := providers[source.Provider]; !exists {
			return nil, fmt.Errorf("unknown provider %q for %s", source.Provider, param.Name)
		}
		sourceable = append(sourceable, param)
	}
	return sourceable, nil
}
