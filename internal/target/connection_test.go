package target_test

import (
	"testing"
	"time"

	"github.com/arm/topo/internal/target"
	"github.com/arm/topo/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func TestConnectionOptions(t *testing.T) {
	t.Run("SSHArgs", func(t *testing.T) {
		t.Run("with multiplexing enabled on non-windows includes Control args", func(t *testing.T) {
			testutil.RequireOS(t, "linux")
			opts := target.ConnectionOptions{Multiplex: true}

			assert.Equal(t, []string{
				"-o", "ControlMaster=auto",
				"-o", "ControlPersist=10s",
				"-o", "ControlPath=~/.ssh/topo-cm-%r@%h:%p",
			}, opts.SSHArgs())
		})

		t.Run("with multiplexing enabled on windows includes no Control args", func(t *testing.T) {
			testutil.RequireOS(t, "windows")
			opts := target.ConnectionOptions{Multiplex: true}

			assert.Nil(t, opts.SSHArgs())
		})

		t.Run("with multiplexing disabled includes no Control args", func(t *testing.T) {
			opts := target.ConnectionOptions{Multiplex: false}

			assert.Nil(t, opts.SSHArgs())
		})

		t.Run("with connect timeout includes ConnectTimeout arg", func(t *testing.T) {
			opts := target.ConnectionOptions{ConnectTimeout: 10 * time.Second}

			assert.Equal(t, []string{"-o", "ConnectTimeout=10"}, opts.SSHArgs())
		})

		t.Run("with no timeout includes no ConnectTimeout arg", func(t *testing.T) {
			opts := target.ConnectionOptions{}

			assert.Nil(t, opts.SSHArgs())
		})
	})
}
