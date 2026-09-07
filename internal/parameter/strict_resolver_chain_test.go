package parameter_test

import (
	"errors"
	"testing"

	"github.com/arm/topo/internal/parameter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockResolver struct {
	mock.Mock
}

func (m *mockResolver) Resolve(definitions []parameter.Definition) (parameter.Values, error) {
	call := m.Called(definitions)
	if call.Get(0) == nil {
		return nil, call.Error(1)
	}
	return call.Get(0).(parameter.Values), call.Error(1)
}

func TestStrictResolverChain(t *testing.T) {
	t.Run("collects from single resolver", func(t *testing.T) {
		resolver := &mockResolver{}
		definitions := []parameter.Definition{
			{Name: "GREETING", Required: true},
		}
		resolver.On("Resolve", definitions).Return(parameter.Values{"GREETING": "Hello"}, nil)
		chain := parameter.NewStrictResolverChain(resolver)

		got, err := chain.Resolve(definitions)

		require.NoError(t, err)
		want := parameter.Values{"GREETING": "Hello"}
		assert.Equal(t, want, got)
		resolver.AssertExpectations(t)
	})

	t.Run("errors when required parameters are missing", func(t *testing.T) {
		resolver := &mockResolver{}
		missing := parameter.Definition{Name: "GREETING", Required: true, Description: "The greeting"}
		definitions := []parameter.Definition{
			missing,
			{Name: "PORT", Required: false},
		}
		resolver.On("Resolve", definitions).Return(parameter.Values{"PORT": "8080"}, nil)
		chain := parameter.NewStrictResolverChain(resolver)

		_, err := chain.Resolve(definitions)

		assert.Equal(t, parameter.MissingParametersError{missing}, err)
		resolver.AssertExpectations(t)
	})

	t.Run("allows missing optional parameters", func(t *testing.T) {
		resolver := &mockResolver{}
		definitions := []parameter.Definition{
			{Name: "GREETING", Required: true},
			{Name: "PORT", Required: false},
		}
		resolver.On("Resolve", definitions).Return(parameter.Values{"GREETING": "Hello"}, nil)
		chain := parameter.NewStrictResolverChain(resolver)

		got, err := chain.Resolve(definitions)

		require.NoError(t, err)
		want := parameter.Values{"GREETING": "Hello"}
		assert.Equal(t, want, got)
		resolver.AssertExpectations(t)
	})

	t.Run("errors when resolver fails", func(t *testing.T) {
		resolver := &mockResolver{}
		definitions := []parameter.Definition{
			{Name: "GREETING", Required: true},
		}
		resolver.On("Resolve", mock.Anything).Return(nil, errors.New("big bang"))
		chain := parameter.NewStrictResolverChain(resolver)

		_, err := chain.Resolve(definitions)

		require.Error(t, err)
		assert.EqualError(t, err, "big bang")
		resolver.AssertExpectations(t)
	})

	t.Run("stops calling resolvers when all required parameters are satisfied", func(t *testing.T) {
		resolver1 := &mockResolver{}
		resolver2 := &mockResolver{}
		definitions := []parameter.Definition{
			{Name: "GREETING", Required: true},
			{Name: "PORT", Required: false},
		}
		resolver1.On("Resolve", definitions).Return(parameter.Values{"GREETING": "Hello"}, nil)
		chain := parameter.NewStrictResolverChain(resolver1, resolver2)

		got, err := chain.Resolve(definitions)

		require.NoError(t, err)
		want := parameter.Values{"GREETING": "Hello"}
		assert.Equal(t, want, got)
		resolver1.AssertExpectations(t)
		resolver2.AssertNotCalled(t, "Resolve")
	})

	t.Run("calls second resolver when first does not satisfy all required parameters", func(t *testing.T) {
		resolver1 := &mockResolver{}
		resolver2 := &mockResolver{}
		all := []parameter.Definition{
			{Name: "GREETING", Required: true},
			{Name: "NAME", Required: true},
			{Name: "PORT", Required: false},
		}
		remaining := []parameter.Definition{
			{Name: "NAME", Required: true},
			{Name: "PORT", Required: false},
		}
		resolver1.On("Resolve", all).Return(parameter.Values{"GREETING": "Hello"}, nil)
		resolver2.On("Resolve", remaining).Return(parameter.Values{"NAME": "World"}, nil)
		chain := parameter.NewStrictResolverChain(resolver1, resolver2)

		got, err := chain.Resolve(all)

		require.NoError(t, err)
		want := parameter.Values{
			"GREETING": "Hello",
			"NAME":     "World",
		}
		assert.Equal(t, want, got)
		resolver1.AssertExpectations(t)
		resolver2.AssertExpectations(t)
	})

	t.Run("collects values from multiple resolvers", func(t *testing.T) {
		resolver1 := parameter.NewStaticResolver(parameter.Values{
			"PORT": "8080",
			"NAME": "Topo",
		})
		resolver2 := parameter.NewStaticResolver(parameter.Values{
			"GREETING": "Hello",
		})
		chain := parameter.NewStrictResolverChain(resolver1, resolver2)
		definitions := []parameter.Definition{
			{Name: "NAME", Required: true},
			{Name: "GREETING", Required: true},
			{Name: "PORT", Required: true},
		}

		got, err := chain.Resolve(definitions)

		require.NoError(t, err)
		want := parameter.Values{
			"NAME":     "Topo",
			"GREETING": "Hello",
			"PORT":     "8080",
		}
		assert.Equal(t, want, got)
	})

	t.Run("allows required parameters with non-empty current values", func(t *testing.T) {
		resolver := parameter.NewStaticResolver(nil)
		chain := parameter.NewStrictResolverChain(resolver)
		definitions := []parameter.Definition{{
			Name:          "CINNAMON",
			Required:      true,
			CurrentValues: []string{"current", "${CINNAMON}"},
		}}

		got, err := chain.Resolve(definitions)

		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("errors when any current value is empty", func(t *testing.T) {
		resolver := parameter.NewStaticResolver(nil)
		chain := parameter.NewStrictResolverChain(resolver)
		definition := parameter.Definition{
			Name:          "CINNAMON",
			Required:      true,
			CurrentValues: []string{"current", ""},
		}

		_, err := chain.Resolve([]parameter.Definition{definition})

		assert.Equal(t, parameter.MissingParametersError{definition}, err)
	})

	t.Run("does not resolve omitted optional parameters", func(t *testing.T) {
		resolver := parameter.NewStaticResolver(nil)
		chain := parameter.NewStrictResolverChain(resolver)
		definitions := []parameter.Definition{
			{
				Name:     "CINNAMON",
				Required: false,
			},
		}

		got, err := chain.Resolve(definitions)

		require.NoError(t, err)
		want := parameter.Values{}
		assert.Equal(t, want, got)
	})
}

func TestMissingParametersError(t *testing.T) {
	t.Run("formats error message with descriptions", func(t *testing.T) {
		err := parameter.MissingParametersError{
			{
				Name:        "GREETING",
				Description: "The greeting message",
				Example:     "Hello",
			},
			{
				Name:          "PORT",
				Description:   "Port number",
				CurrentValues: []string{"8080", ""},
			},
		}

		got := err.Error()

		want := `missing value(s) for required parameters:
  GREETING:
    description: The greeting message
    example: Hello
  PORT:
    description: Port number
    # current: ["8080",""]
`
		assert.Equal(t, want, got)
	})
}
