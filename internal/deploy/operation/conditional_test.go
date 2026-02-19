package operation_test

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/arm/topo/internal/deploy/operation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockOperation struct {
	mock.Mock
}

func (m *mockOperation) Description() string {
	args := m.Called()
	return args.String(0)
}

func (m *mockOperation) Run(cmdOutput io.Writer) error {
	args := m.Called(cmdOutput)
	return args.Error(0)
}

func (m *mockOperation) DryRun(output io.Writer) error {
	args := m.Called(output)
	return args.Error(0)
}

type mockPredicate struct {
	result bool
}

func (p *mockPredicate) Eval() bool {
	return p.result
}

func TestConditional(t *testing.T) {
	t.Run("Run", func(t *testing.T) {
		t.Run("executes ifTrue operation when condition is true", func(t *testing.T) {
			ifTrue := new(mockOperation)
			condition := &mockPredicate{result: true}
			var buf bytes.Buffer
			ifTrue.On("Run", &buf).Return(nil)
			op := operation.NewConditional(condition, ifTrue, new(mockOperation))

			err := op.Run(&buf)

			require.NoError(t, err)
			ifTrue.AssertExpectations(t)
		})

		t.Run("executes ifFalse operation when condition is false", func(t *testing.T) {
			ifFalse := new(mockOperation)
			condition := &mockPredicate{result: false}
			var buf bytes.Buffer
			ifFalse.On("Run", &buf).Return(nil)
			op := operation.NewConditional(condition, new(mockOperation), ifFalse)

			err := op.Run(&buf)

			require.NoError(t, err)
			ifFalse.AssertExpectations(t)
		})

		t.Run("returns error from ifTrue operation", func(t *testing.T) {
			expectedErr := errors.New("ifTrue error")
			ifTrue := new(mockOperation)
			condition := &mockPredicate{result: true}
			var buf bytes.Buffer
			ifTrue.On("Run", &buf).Return(expectedErr)
			op := operation.NewConditional(condition, ifTrue, new(mockOperation))

			err := op.Run(&buf)

			assert.Equal(t, expectedErr, err)
			ifTrue.AssertExpectations(t)
		})

		t.Run("returns error from ifFalse operation", func(t *testing.T) {
			expectedErr := errors.New("ifFalse error")
			ifFalse := new(mockOperation)
			condition := &mockPredicate{result: false}
			var buf bytes.Buffer
			ifFalse.On("Run", &buf).Return(expectedErr)
			op := operation.NewConditional(condition, new(mockOperation), ifFalse)

			err := op.Run(&buf)

			assert.Equal(t, expectedErr, err)
			ifFalse.AssertExpectations(t)
		})
	})

	t.Run("DryRun", func(t *testing.T) {
		t.Run("executes ifTrue DryRun when condition is true", func(t *testing.T) {
			ifTrue := new(mockOperation)
			condition := &mockPredicate{result: true}
			var buf bytes.Buffer
			ifTrue.On("DryRun", &buf).Return(nil)
			op := operation.NewConditional(condition, ifTrue, new(mockOperation))

			err := op.DryRun(&buf)

			require.NoError(t, err)
			ifTrue.AssertExpectations(t)
		})

		t.Run("executes ifFalse DryRun when condition is false", func(t *testing.T) {
			ifFalse := new(mockOperation)
			condition := &mockPredicate{result: false}
			var buf bytes.Buffer
			ifFalse.On("DryRun", &buf).Return(nil)
			op := operation.NewConditional(condition, new(mockOperation), ifFalse)

			err := op.DryRun(&buf)

			require.NoError(t, err)
			ifFalse.AssertExpectations(t)
		})

		t.Run("returns error from ifTrue DryRun", func(t *testing.T) {
			expectedErr := errors.New("ifTrue dryrun error")
			ifTrue := new(mockOperation)
			condition := &mockPredicate{result: true}
			var buf bytes.Buffer
			ifTrue.On("DryRun", &buf).Return(expectedErr)
			op := operation.NewConditional(condition, ifTrue, new(mockOperation))

			err := op.DryRun(&buf)

			assert.Equal(t, expectedErr, err)
			ifTrue.AssertExpectations(t)
		})

		t.Run("returns error from ifFalse DryRun", func(t *testing.T) {
			expectedErr := errors.New("ifFalse dryrun error")
			ifFalse := new(mockOperation)
			condition := &mockPredicate{result: false}
			var buf bytes.Buffer
			ifFalse.On("DryRun", &buf).Return(expectedErr)
			op := operation.NewConditional(condition, new(mockOperation), ifFalse)

			err := op.DryRun(&buf)

			assert.Equal(t, expectedErr, err)
			ifFalse.AssertExpectations(t)
		})
	})

	t.Run("Description", func(t *testing.T) {
		t.Run("returns ifTrue description when condition is true", func(t *testing.T) {
			ifTrue := new(mockOperation)
			condition := &mockPredicate{result: true}
			ifTrue.On("Description").Return("ifTrue description")
			op := operation.NewConditional(condition, ifTrue, new(mockOperation))

			got := op.Description()

			assert.Equal(t, ifTrue.Description(), got)
		})

		t.Run("returns ifFalse description when condition is false", func(t *testing.T) {
			ifFalse := new(mockOperation)
			condition := &mockPredicate{result: false}
			ifFalse.On("Description").Return("ifFalse description")
			op := operation.NewConditional(condition, new(mockOperation), ifFalse)

			got := op.Description()

			assert.Equal(t, ifFalse.Description(), got)
		})
	})
}
