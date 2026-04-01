package runner

import "github.com/stretchr/testify/mock"

// Mock implements Runner.
// Use it in tests to stub runner behaviour without rolling a custom fake.
type Mock struct {
	mock.Mock
}

func (m *Mock) Run(command string) (string, error) {
	args := m.Called(command)
	return args.String(0), args.Error(1)
}

func (m *Mock) RunWithStdin(command string, stdin []byte) (string, error) {
	args := m.Called(command, stdin)
	return args.String(0), args.Error(1)
}
