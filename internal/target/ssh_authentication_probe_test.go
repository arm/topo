package target_test

import (
	"testing"

	"github.com/arm/topo/internal/target"
	"github.com/stretchr/testify/assert"
)

func TestBuildProbeArgs(t *testing.T) {
	tests := []struct {
		name              string
		acceptNewHostKeys bool
		want              []string
	}{
		{
			name: "strict host key checking by default",
			want: []string{
				"-o", "BatchMode=yes",
				"-o", "PreferredAuthentications=publickey",
				"-o", "PasswordAuthentication=no",
				"-o", "NumberOfPasswordPrompts=0",
				"-o", "StrictHostKeyChecking=yes",
			},
		},
		{
			name:              "accept new host keys",
			acceptNewHostKeys: true,
			want: []string{
				"-o", "BatchMode=yes",
				"-o", "PreferredAuthentications=publickey",
				"-o", "PasswordAuthentication=no",
				"-o", "NumberOfPasswordPrompts=0",
				"-o", "StrictHostKeyChecking=accept-new",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := target.BuildProbeArgs(tt.acceptNewHostKeys)

			assert.Equal(t, tt.want, got)
		})
	}
}
