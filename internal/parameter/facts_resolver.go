package parameter

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"unicode"

	"github.com/arm/topo/internal/fact"
	"github.com/arm/topo/internal/output/logger"
	"github.com/arm/topo/internal/runner"
	"github.com/arm/topo/internal/ssh"
)

type FactsResolver struct {
	providers map[string]fact.Provider
	ctx       context.Context
	target    string
}

func NewFactsResolver(ctx context.Context, target string, providers map[string]fact.Provider) *FactsResolver {
	return &FactsResolver{ctx: ctx, target: target, providers: providers}
}

func (r *FactsResolver) Resolve(parameters []Parameter) (Changes, error) {
	bindings, err := r.parseBindings(parameters)
	if err != nil {
		return nil, err
	}

	if r.target == "" {
		for _, binding := range bindings {
			logger.Warn(fmt.Sprintf("cannot automatically configure parameter %q without a target; provide a target or supply this parameter manually", binding.parameterName))
		}
		return nil, nil
	}

	runner := runner.For(ssh.NewDestination(r.target))

	changes := Changes{}
	results := map[string]fact.Report{}
	for _, binding := range bindings {
		result, ok := results[binding.provider]
		if !ok {
			var err error
			result, err = r.providers[binding.provider].Facts(r.ctx, runner)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve %s from %q: %w", binding.parameterName, binding.source, err)
			}
			if err := result.Validate(); err != nil {
				return nil, fmt.Errorf("failed to resolve %s from %q: %w", binding.parameterName, binding.source, err)
			}
			results[binding.provider] = result
		}
		fact, ok := result[binding.key]
		if !ok {
			return nil, fmt.Errorf("target fact %q not found for %s", binding.source, binding.parameterName)
		}
		changes[binding.parameterName] = new(fact.Value.String())
	}
	return changes, nil
}

type factBinding struct {
	parameterName string
	source        string
	provider      string
	key           string
}

func (r *FactsResolver) parseBindings(parameters []Parameter) ([]factBinding, error) {
	var bindings []factBinding
	for _, param := range parameters {
		if param.Source == nil {
			continue
		}
		binding, err := r.parseBinding(param.Name, *param.Source)
		if err != nil {
			return nil, err
		}
		bindings = append(bindings, binding)
	}
	return bindings, nil
}

func (r *FactsResolver) parseBinding(parameterName, source string) (factBinding, error) {
	parts := strings.Split(source, ".")
	if len(parts) < 2 || slices.Contains(parts, "") || strings.ContainsFunc(source, unicode.IsSpace) {
		return factBinding{}, fmt.Errorf("invalid source %q for %s", source, parameterName)
	}

	binding := factBinding{parameterName: parameterName, source: source}
	for provider := range r.providers {
		key, matches := strings.CutPrefix(source, provider+".")
		if !matches {
			continue
		}
		if binding.provider != "" {
			return factBinding{}, fmt.Errorf("multiple fact providers match source %q for %s", source, parameterName)
		}
		binding.provider = provider
		binding.key = key
	}
	if binding.provider == "" {
		return factBinding{}, fmt.Errorf("no fact provider matches source %q for %s", source, parameterName)
	}
	return binding, nil
}
