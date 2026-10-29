package parameter_test

import (
	"context"
	"testing"

	"github.com/arm/topo/internal/fact"
	"github.com/arm/topo/internal/parameter"
	"github.com/arm/topo/internal/runner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeFactsProvider struct {
	result fact.Report
	err    error
	calls  int
}

func (p *fakeFactsProvider) Facts(context.Context, runner.CommandRunner) (fact.Report, error) {
	p.calls++
	return p.result, p.err
}

func TestFactsResolver(t *testing.T) {
	t.Run("resolves facts using one call per provider", func(t *testing.T) {
		provider := &fakeFactsProvider{result: fact.Report{Groups: []fact.Group{{
			ID: "settings",
			Facts: []fact.Fact{
				{Key: "first", Value: fact.Value{Type: "integer", Integer: 42}},
				{Key: "second", Value: fact.Value{Type: "integer", Integer: 24}},
			},
		}}}}
		resolver := parameter.NewFactsResolver(context.Background(), "localhost", map[string]fact.Provider{"test": provider})
		parameters := []parameter.Parameter{
			{Name: "FIRST", Source: &parameter.Source{Provider: "test", Group: "settings", Fact: "first"}},
			{Name: "SECOND", Source: &parameter.Source{Provider: "test", Group: "settings", Fact: "second"}},
		}

		got, err := resolver.Resolve(parameters)

		require.NoError(t, err)
		assert.Equal(t, parameter.Changes{"FIRST": new("42"), "SECOND": new("24")}, got)
		assert.Equal(t, 1, provider.calls)
	})

	t.Run("CLI assignments and removals override fact sources", func(t *testing.T) {
		provider := &fakeFactsProvider{err: assert.AnError}
		cliResolver, err := parameter.NewCLIResolver([]string{"OVERRIDE=manual"}, []string{"REMOVE"})
		require.NoError(t, err)
		factsResolver := parameter.NewFactsResolver(context.Background(), "localhost", map[string]fact.Provider{"test": provider})
		resolver := parameter.NewStrictResolverChain(cliResolver, factsResolver)
		parameters := []parameter.Parameter{
			{Name: "OVERRIDE", Source: &parameter.Source{Provider: "test", Group: "g", Fact: "k"}},
			{Name: "REMOVE", Source: &parameter.Source{Provider: "test", Group: "g", Fact: "k"}},
		}

		got, err := resolver.Resolve(parameters)

		require.NoError(t, err)
		assert.Equal(t, parameter.Changes{"OVERRIDE": new("manual"), "REMOVE": nil}, got)
		assert.Zero(t, provider.calls)
	})

	t.Run("skips fact collection without a target", func(t *testing.T) {
		provider := &fakeFactsProvider{err: assert.AnError}
		resolver := parameter.NewFactsResolver(context.Background(), "", map[string]fact.Provider{"test": provider})
		parameters := []parameter.Parameter{
			{Name: "VALUE", Source: &parameter.Source{Provider: "test", Group: "g", Fact: "k"}},
		}

		got, err := resolver.Resolve(parameters)

		require.NoError(t, err)
		assert.Empty(t, got)
		assert.Zero(t, provider.calls)
	})

	t.Run("ignores parameters without a source", func(t *testing.T) {
		provider := &fakeFactsProvider{err: assert.AnError}
		resolver := parameter.NewFactsResolver(context.Background(), "localhost", map[string]fact.Provider{"test": provider})

		got, err := resolver.Resolve([]parameter.Parameter{{Name: "MANUAL"}})

		require.NoError(t, err)
		assert.Empty(t, got)
		assert.Zero(t, provider.calls)
	})

	t.Run("reports source and provider errors", func(t *testing.T) {
		for _, tc := range []struct {
			name     string
			source   parameter.Source
			provider fakeFactsProvider
			want     string
		}{
			{name: "incomplete source", source: parameter.Source{Provider: "test"}, want: "requires provider, group and fact"},
			{name: "unknown provider", source: parameter.Source{Provider: "unknown", Group: "g", Fact: "k"}, want: "unknown provider"},
			{name: "missing fact", source: parameter.Source{Provider: "test", Group: "g", Fact: "k"}, want: "not found in group"},
			{name: "probe failure", source: parameter.Source{Provider: "test", Group: "g", Fact: "k"}, provider: fakeFactsProvider{err: assert.AnError}, want: assert.AnError.Error()},
			{name: "invalid result", source: parameter.Source{Provider: "test", Group: "g", Fact: "k"}, provider: fakeFactsProvider{result: fact.Report{Groups: []fact.Group{{}}}}, want: "empty or duplicate fact group"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				resolver := parameter.NewFactsResolver(context.Background(), "localhost", map[string]fact.Provider{"test": &tc.provider})

				got, err := resolver.Resolve([]parameter.Parameter{{Name: "VALUE", Source: &tc.source}})

				require.ErrorContains(t, err, tc.want)
				assert.ErrorContains(t, err, "VALUE")
				assert.Nil(t, got)
			})
		}
	})
}

func TestSourceableParameters(t *testing.T) {
	t.Run("filters out parameters without a source", func(t *testing.T) {
		bound := parameter.Parameter{Name: "BOUND", Source: &parameter.Source{Provider: "test", Group: "g", Fact: "k"}}
		providers := map[string]fact.Provider{"test": &fakeFactsProvider{}}

		got, err := parameter.SourceableParameters([]parameter.Parameter{{Name: "MANUAL"}, bound}, providers)

		require.NoError(t, err)
		assert.Equal(t, []parameter.Parameter{bound}, got)
	})

	t.Run("rejects invalid bindings", func(t *testing.T) {
		for _, source := range []parameter.Source{
			{Group: "g", Fact: "k"},
			{Provider: "test", Fact: "k"},
			{Provider: "test", Group: "g"},
			{Provider: "unknown", Group: "g", Fact: "k"},
		} {
			providers := map[string]fact.Provider{"test": &fakeFactsProvider{}}

			got, err := parameter.SourceableParameters([]parameter.Parameter{{Name: "VALUE", Source: &source}}, providers)

			require.ErrorContains(t, err, "VALUE")
			assert.Nil(t, got)
		}
	})
}
