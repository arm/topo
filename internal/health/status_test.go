package health_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/arm/topo/internal/health"
	"github.com/arm/topo/internal/target"
	"github.com/stretchr/testify/assert"
)

type fakeRunner struct {
	run func(command string) (string, error)
}

func (f fakeRunner) Run(command string) (string, error) {
	return f.run(command)
}

func TestProbeHealthStatus(t *testing.T) {
	t.Run("ProbeHealthStatus", func(t *testing.T) {
		t.Run("finds remote CPUs", func(t *testing.T) {
			r := fakeRunner{run: func(command string) (string, error) {
				switch {
				case strings.Contains(command, "ls /sys/class/remoteproc"):
					return "remoteproc0\nremoteproc1", nil
				case strings.Contains(command, "cat /sys/class/remoteproc"):
					return "foo\nbar", nil
				default:
					return "", fmt.Errorf("unexpected command: %s", command)
				}
			}}

			ts := health.ProbeHealthStatus(r)

			want := health.HardwareProfile{RemoteCPU: []target.RemoteprocCPU{{Name: "foo"}, {Name: "bar"}}}
			assert.Equal(t, want, ts.Hardware)
		})

		t.Run("succeeds when no remoteproc support", func(t *testing.T) {
			r := fakeRunner{run: func(command string) (string, error) {
				return "", fmt.Errorf("no such directory")
			}}

			ts := health.ProbeHealthStatus(r)

			assert.Len(t, ts.Hardware.RemoteCPU, 0)
		})
	})
}
