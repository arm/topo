package parameter_test

import (
	"context"
	"fmt"
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
		provider := &fakeFactsProvider{result: fact.Report{
			"slot0.label": {Value: fact.Value{Type: "string", Text: "first"}},
			"slot1.label": {Value: fact.Value{Type: "string", Text: "second"}},
		}}
		resolver := parameter.NewFactsResolver(context.Background(), "localhost", map[string]fact.Provider{"example.target.inventory": provider})
		parameters := []parameter.Parameter{
			{Name: "FIRST", Source: new("example.target.inventory.slot0.label")},
			{Name: "SECOND", Source: new("example.target.inventory.slot1.label")},
		}

		got, err := resolver.Resolve(parameters)

		require.NoError(t, err)
		assert.Equal(t, parameter.Changes{"FIRST": new("first"), "SECOND": new("second")}, got)
		assert.Equal(t, 1, provider.calls)
	})

	t.Run("rejects a source that matches more than one provider", func(t *testing.T) {
		provider := &fakeFactsProvider{}
		resolver := parameter.NewFactsResolver(context.Background(), "localhost", map[string]fact.Provider{
			"example.target.inventory":       provider,
			"example.target.inventory.slot0": provider,
		})

		got, err := resolver.Resolve([]parameter.Parameter{
			{Name: "NAME", Source: new("example.target.inventory.slot0.label")},
		})

		require.ErrorContains(t, err, "multiple fact providers match source")
		assert.Nil(t, got)
		assert.Zero(t, provider.calls)
	})

	t.Run("CLI assignments and removals override fact sources", func(t *testing.T) {
		provider := &fakeFactsProvider{err: assert.AnError}
		cliResolver, err := parameter.NewCLIResolver([]string{"OVERRIDE=manual"}, []string{"REMOVE"})
		require.NoError(t, err)
		factsResolver := parameter.NewFactsResolver(context.Background(), "localhost", map[string]fact.Provider{"example.target.inventory": provider})
		resolver := parameter.NewStrictResolverChain(cliResolver, factsResolver)
		parameters := []parameter.Parameter{
			{Name: "OVERRIDE", Source: new("example.target.inventory.capacity")},
			{Name: "REMOVE", Source: new("example.target.inventory.capacity")},
		}

		got, err := resolver.Resolve(parameters)

		require.NoError(t, err)
		assert.Equal(t, parameter.Changes{"OVERRIDE": new("manual"), "REMOVE": nil}, got)
		assert.Zero(t, provider.calls)
	})

	t.Run("skips fact collection without a target", func(t *testing.T) {
		provider := &fakeFactsProvider{err: assert.AnError}
		resolver := parameter.NewFactsResolver(context.Background(), "", map[string]fact.Provider{"example.target.inventory": provider})
		parameters := []parameter.Parameter{
			{Name: "VALUE", Source: new("example.target.inventory.capacity")},
		}

		got, err := resolver.Resolve(parameters)

		require.NoError(t, err)
		assert.Empty(t, got)
		assert.Zero(t, provider.calls)
	})

	t.Run("ignores parameters without a source", func(t *testing.T) {
		provider := &fakeFactsProvider{err: assert.AnError}
		resolver := parameter.NewFactsResolver(context.Background(), "localhost", map[string]fact.Provider{"example.target.inventory": provider})

		got, err := resolver.Resolve([]parameter.Parameter{{Name: "MANUAL"}})

		require.NoError(t, err)
		assert.Empty(t, got)
		assert.Zero(t, provider.calls)
	})

	t.Run("reports source and provider errors", func(t *testing.T) {
		for _, tc := range []struct {
			name     string
			source   string
			provider fakeFactsProvider
			want     string
		}{
			{name: "empty source", want: "invalid source"},
			{name: "empty path segment", source: "example.target.inventory..capacity", want: "invalid source"},
			{name: "too few segments", source: "inventory", want: "invalid source"},
			{name: "unknown provider", source: "example.target.unknown.capacity", want: "no fact provider matches source"},
			{name: "provider namespace differs", source: "example.host.inventory.capacity", want: "no fact provider matches source"},
			{name: "missing fact", source: "example.target.inventory.capacity", want: "not found"},
			{name: "probe failure", source: "example.target.inventory.capacity", provider: fakeFactsProvider{err: assert.AnError}, want: assert.AnError.Error()},
			{name: "invalid result", source: "example.target.inventory.capacity", provider: fakeFactsProvider{result: fact.Report{"": {}}}, want: "invalid fact key"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				resolver := parameter.NewFactsResolver(context.Background(), "localhost", map[string]fact.Provider{"example.target.inventory": &tc.provider})

				got, err := resolver.Resolve([]parameter.Parameter{{Name: "VALUE", Source: &tc.source}})

				require.ErrorContains(t, err, tc.want)
				assert.ErrorContains(t, err, "VALUE")
				assert.ErrorContains(t, err, fmt.Sprintf("%q", tc.source))
				assert.Nil(t, got)
			})
		}
	})
}
