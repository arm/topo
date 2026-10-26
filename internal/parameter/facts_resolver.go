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
	changes := Changes{}
	results := map[string]fact.Report{}
	for _, parameter := range parameters {
		source := parameter.Source
		if source == nil {
			continue
		}
		if source.Provider == "" || source.Group == "" || source.Select == "" {
			return nil, fmt.Errorf("source requires provider, group and select for %s", parameter.Name)
		}
		result, ok := results[source.Provider]
		if !ok {
			provider, exists := r.providers[source.Provider]
			if !exists {
				return nil, fmt.Errorf("unknown provider %q for %s", source.Provider, parameter.Name)
			}
			var err error
			result, err = provider.Facts(r.ctx, r.runner)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve %s: %w", parameter.Name, err)
			}
			if err := result.Validate(); err != nil {
				return nil, fmt.Errorf("failed to resolve %s: %w", parameter.Name, err)
			}
			results[source.Provider] = result
		}
		fact, ok := result.Find(source.Group, source.Select)
		if !ok {
			return nil, fmt.Errorf("fact %q not found in group %q for %s", source.Select, source.Group, parameter.Name)
		}
		changes[parameter.Name] = new(fact.Value.String())
	}
	return changes, nil
}
