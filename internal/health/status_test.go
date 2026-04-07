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
		r.On("Run", context.Background(), mock.AnythingOfType("string")).Maybe().Return("", fmt.Errorf("not found"))
		r.On("BinaryExists", mock.AnythingOfType("string")).Maybe().Return(fmt.Errorf("not found"))

		ts := health.ProbeHealthStatus(context.Background(), r)

		want := health.HardwareProfile{RemoteCPU: []target.RemoteprocCPU{{Name: "foo"}, {Name: "bar"}}}
		assert.Equal(t, want, ts.Hardware)
		r.AssertExpectations(t)
	})

	t.Run("reports binary name when lookup fails", func(t *testing.T) {
		r := &runner.Mock{}
		r.On("Run", context.Background(), command.WrapInLoginShell("ls /sys/class/remoteproc")).Return("", fmt.Errorf("not found"))
		r.On("BinaryExists", "docker").Return(fmt.Errorf(`"docker" not found in $PATH`))
		r.On("BinaryExists", "lscpu").Return(fmt.Errorf(`"lscpu" not found in $PATH`))

		ts := health.ProbeHealthStatus(context.Background(), r)

		for _, dep := range ts.Dependencies {
			if dep.Error != nil {
				assert.Contains(t, dep.Error.Error(), fmt.Sprintf("%q", dep.Dependency.Binary))
			}
		}
	})

	t.Run("succeeds when no remoteproc support", func(t *testing.T) {
		r := &runner.Mock{}
		r.On("Run", context.Background(), mock.AnythingOfType("string")).Return("", fmt.Errorf("no such directory"))
		r.On("BinaryExists", mock.AnythingOfType("string")).Maybe().Return(fmt.Errorf("not found"))

		ts := health.ProbeHealthStatus(context.Background(), r)

		assert.Len(t, ts.Hardware.RemoteCPU, 0)
		r.AssertExpectations(t)
	})
}
