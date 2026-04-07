package health_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/arm/topo/internal/command"
	"github.com/arm/topo/internal/health"
	"github.com/arm/topo/internal/runner"
	"github.com/arm/topo/internal/target"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestProbeHealthStatus(t *testing.T) {
	t.Run("finds remote CPUs", func(t *testing.T) {
		r := &runner.Mock{}
		r.On("Run", context.Background(), command.WrapInLoginShell("ls /sys/class/remoteproc")).Return("remoteproc0\nremoteproc1", nil)
		r.On("Run", context.Background(), command.WrapInLoginShell("cat /sys/class/remoteproc/*/name")).Return("foo\nbar", nil)
		r.On("BinaryExists", mock.AnythingOfType("string")).Maybe().Return(fmt.Errorf("not found"))

		ts := health.ProbeHealthStatus(context.Background(), r)

		want := health.HardwareProfile{RemoteCPU: []target.RemoteprocCPU{{Name: "foo"}, {Name: "bar"}}}
		assert.Equal(t, want, ts.Hardware)
		r.AssertExpectations(t)
	})

	t.Run("succeeds when no remoteproc support", func(t *testing.T) {
		r := &runner.Mock{}
		r.On("Run", context.Background(), command.WrapInLoginShell("ls /sys/class/remoteproc")).Return("", fmt.Errorf("no such directory"))
		r.On("BinaryExists", mock.AnythingOfType("string")).Maybe().Return(fmt.Errorf("not found"))

		ts := health.ProbeHealthStatus(context.Background(), r)

		assert.Len(t, ts.Hardware.RemoteCPU, 0)
		r.AssertExpectations(t)
	})
}
