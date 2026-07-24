package netprobe_test

import (
	"strconv"
	"testing"

	"github.com/arm/topo/internal/netprobe"
	"github.com/stretchr/testify/assert"
)

const (
	procNetTCPV4 = `  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode
   0: 0100007F:2AF8 00000000:0000 0A 00000000:00000000 00:00000000 00000000   117        0 18993 2 0000000000000000 100 0 0 10 0
   1: 00000000:2CAA 00000000:0000 0A 00000000:00000000 00:00000000 00000000     0        0 150348294 1 0000000000000000 100 0 0 10 0`

	procNetTCPV6 = `  sl  local_address                         remote_address                        st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode
   0: 00000000000000000000000000000000:2CAA 00000000000000000000000000000000:0000 0A 00000000:00000000 00:00000000 00000000     0        0 150348295 1 0000000000000000 100 0 0 10 0
  10: 00000000000000000000000001000000:0277 00000000000000000000000000000000:0000 0A 00000000:00000000 00:00000000 00000000     0        0 160432833 1 0000000000000000 100 0 0 10 0`
)

func TestIsPortListeningInProcNetTCP(t *testing.T) {
	tests := []struct {
		name       string
		procNetTCP string
		port       string
		expected   bool
	}{
		{
			name:       "returns false for an IPv4 loopback listener",
			procNetTCP: procNetTCPV4,
			port:       strconv.Itoa(0x2AF8),
			expected:   false,
		},
		{
			name:       "returns true for an IPv4 all-interfaces listener",
			procNetTCP: procNetTCPV4,
			port:       strconv.Itoa(0x2CAA),
			expected:   true,
		},
		{
			name:       "returns false for an IPv6 loopback listener",
			procNetTCP: procNetTCPV6,
			port:       strconv.Itoa(0x0277),
			expected:   false,
		},
		{
			name:       "returns true for an IPv6 all-interfaces listener",
			procNetTCP: procNetTCPV6,
			port:       strconv.Itoa(0x2CAA),
			expected:   true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := netprobe.IsRemotePortListeningInProcNetTCP(test.procNetTCP, test.port)

			assert.NoError(t, err)
			assert.Equal(t, test.expected, got)
		})
	}

	t.Run("returns false when the port is not present", func(t *testing.T) {
		got, err := netprobe.IsRemotePortListeningInProcNetTCP(procNetTCPV4, "9999")

		assert.NoError(t, err)
		assert.False(t, got)
	})

	t.Run("returns false when the matching socket is not listening", func(t *testing.T) {
		procNetTCP := `sl local_address rem_address st
0: 0100007F:2AF8 00000000:0000 01`
		port := strconv.Itoa(0x2AF8)

		got, err := netprobe.IsRemotePortListeningInProcNetTCP(procNetTCP, port)

		assert.NoError(t, err)
		assert.False(t, got)
	})

	t.Run("returns true when a non-loopback interface is listening", func(t *testing.T) {
		procNetTCP := `sl local_address rem_address st
0: 0100000A:2B28 00000000:0000 0A`
		port := strconv.Itoa(0x2B28)

		got, err := netprobe.IsRemotePortListeningInProcNetTCP(procNetTCP, port)

		assert.NoError(t, err)
		assert.True(t, got)
	})

	t.Run("returns an error for an invalid port", func(t *testing.T) {
		procNetTCP := `sl local_address rem_address st`
		got, err := netprobe.IsRemotePortListeningInProcNetTCP(procNetTCP, "not-a-port")

		assert.ErrorContains(t, err, `invalid TCP port "not-a-port"`)
		assert.False(t, got)
	})

	t.Run("returns an error when the table is missing required columns", func(t *testing.T) {
		procNetTCP := `sl rem_address
0: 00000000:0000`

		got, err := netprobe.IsRemotePortListeningInProcNetTCP(procNetTCP, "12345")

		assert.ErrorContains(t, err, `missing "local_address" column`)
		assert.False(t, got)
	})

	t.Run("returns an error when a table row is incomplete", func(t *testing.T) {
		procNetTCP := `sl local_address rem_address st
0: 0100007F:1A18`
		port := strconv.Itoa(0x1A18)

		got, err := netprobe.IsRemotePortListeningInProcNetTCP(procNetTCP, port)

		assert.ErrorContains(t, err, "row 2 has 2 fields, expected at least 4")
		assert.False(t, got)
	})
}
