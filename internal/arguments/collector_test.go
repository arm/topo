package arguments

import (
	"errors"
	"testing"

	"github.com/arm-debug/topo-cli/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockProvider struct {
	mock.Mock
}

func (m *mockProvider) Provide(specs []service.ArgSpec) (map[string]string, error) {
	args := m.Called(specs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]string), args.Error(1)
}

func (m *mockProvider) Name() string {
	args := m.Called()
	return args.String(0)
}

func TestCollector(t *testing.T) {
	t.Run("collects from single provider", func(t *testing.T) {
		provider := &mockProvider{}
		specs := []service.ArgSpec{
			{Name: "GREETING", Required: true},
		}
		provider.On("Provide", specs).Return(map[string]string{"GREETING": "Hello"}, nil)
		collector := NewCollector(provider)

		got, err := collector.Collect(specs)

		require.NoError(t, err)
		assert.Equal(t, "Hello", got["GREETING"])
		provider.AssertExpectations(t)
	})

	t.Run("errors when required arguments missing", func(t *testing.T) {
		provider := &mockProvider{}
		missingArg := service.ArgSpec{Name: "GREETING", Required: true, Description: "The greeting"}
		specs := []service.ArgSpec{
			missingArg,
			{Name: "PORT", Required: false},
		}
		provider.On("Provide", specs).Return(map[string]string{"PORT": "8080"}, nil)
		collector := NewCollector(provider)

		_, err := collector.Collect(specs)

		assert.Equal(t, MissingArgsError{missingArg}, err)
		provider.AssertExpectations(t)
	})

	t.Run("allows missing optional arguments", func(t *testing.T) {
		provider := &mockProvider{}
		specs := []service.ArgSpec{
			{Name: "GREETING", Required: true},
			{Name: "PORT", Required: false},
		}
		provider.On("Provide", specs).Return(map[string]string{"GREETING": "Hello"}, nil)
		collector := NewCollector(provider)

		got, err := collector.Collect(specs)

		require.NoError(t, err)
		assert.Equal(t, "Hello", got["GREETING"])
		assert.Empty(t, got["PORT"])
		provider.AssertExpectations(t)
	})

	t.Run("errors when provider fails", func(t *testing.T) {
		provider := &mockProvider{}
		specs := []service.ArgSpec{}
		provider.On("Name").Return("fancy")
		provider.On("Provide", specs).Return(nil, errors.New("big bang"))
		collector := NewCollector(provider)

		_, err := collector.Collect(specs)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "fancy provider failed: big bang")
		provider.AssertExpectations(t)
	})

	t.Run("stops calling providers when all required args satisfied", func(t *testing.T) {
		provider1 := &mockProvider{}
		provider2 := &mockProvider{}
		specs := []service.ArgSpec{
			{Name: "GREETING", Required: true},
			{Name: "PORT", Required: false},
		}
		provider1.On("Provide", specs).Return(map[string]string{"GREETING": "Hello"}, nil)
		collector := NewCollector(provider1, provider2)

		got, err := collector.Collect(specs)

		require.NoError(t, err)
		assert.Equal(t, "Hello", got["GREETING"])
		provider1.AssertExpectations(t)
		provider2.AssertNotCalled(t, "Provide")
	})

	t.Run("calls second provider when first does not satisfy all required args", func(t *testing.T) {
		provider1 := &mockProvider{}
		provider2 := &mockProvider{}
		specs := []service.ArgSpec{
			{Name: "GREETING", Required: true},
			{Name: "NAME", Required: true},
		}
		provider1.On("Provide", specs).Return(map[string]string{"GREETING": "Hello"}, nil)
		provider2.On("Provide", specs).Return(map[string]string{"NAME": "World"}, nil)
		collector := NewCollector(provider1, provider2)

		got, err := collector.Collect(specs)

		require.NoError(t, err)
		assert.Equal(t, "Hello", got["GREETING"])
		assert.Equal(t, "World", got["NAME"])
		provider1.AssertExpectations(t)
		provider2.AssertExpectations(t)
	})
}

func TestMissingArgsError(t *testing.T) {
	t.Run("formats error message with descriptions", func(t *testing.T) {
		err := MissingArgsError{
			{
				Name:        "GREETING",
				Description: "The greeting message",
				Example:     "Hello",
			},
			{
				Name:        "PORT",
				Description: "Port number",
			},
		}

		got := err.Error()

		want := `missing required build arguments:
  GREETING:
    description: The greeting message
    example: Hello
  PORT:
    description: Port number
`
		assert.Equal(t, want, got)
	})
}
