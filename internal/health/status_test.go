package health_test

import (
	"fmt"
	"testing"

	"github.com/arm/topo/internal/command"
	"github.com/arm/topo/internal/health"
	"github.com/arm/topo/internal/target"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockRunner struct {
	mock.Mock
}

func (m *mockRunner) Run(cmd string) (string, error) {
	args := m.Called(cmd)
	return args.String(0), args.Error(1)
}

func TestProbeHealthStatus(t *testing.T) {
	t.Run("ProbeHealthStatus", func(t *testing.T) {
		t.Run("finds remote CPUs", func(t *testing.T) {
			r := new(mockRunner)
			r.On("Run", command.WrapInLoginShell("ls /sys/class/remoteproc")).Return("remoteproc0\nremoteproc1", nil)
			r.On("Run", command.WrapInLoginShell("cat /sys/class/remoteproc/*/name")).Return("foo\nbar", nil)
			r.On("Run", mock.AnythingOfType("string")).Maybe().Return("", fmt.Errorf("not found"))

			ts := health.ProbeHealthStatus(r)

			want := health.HardwareProfile{RemoteCPU: []target.RemoteprocCPU{{Name: "foo"}, {Name: "bar"}}}
			assert.Equal(t, want, ts.Hardware)
			r.AssertExpectations(t)
		})

		t.Run("succeeds when no remoteproc support", func(t *testing.T) {
			r := new(mockRunner)
			r.On("Run", mock.AnythingOfType("string")).Return("", fmt.Errorf("no such directory"))

			ts := health.ProbeHealthStatus(r)

			assert.Len(t, ts.Hardware.RemoteCPU, 0)
			r.AssertExpectations(t)
		})
	})
}
