package install

import (
	"errors"
	"testing"

	"github.com/arm/topo/internal/runner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInstallToFirstWriteableDirFallsBackOnPermissionErrorFromStderr(t *testing.T) {
	paths := []PathCandidate{
		{Path: "/unwritable"},
		{Path: "/writable", OnPath: true},
	}
	binaries := map[string][]byte{"topo": []byte("binary")}
	r := &runner.Fake{Commands: map[string]runner.FakeResult{
		"install -D -m 0755 /dev/stdin /unwritable/topo": {
			Stderr: "install: cannot create regular file '/unwritable/topo': Permission denied",
			Err:    errors.New("exit status 1"),
		},
		"install -D -m 0755 /dev/stdin /writable/topo": {},
	}}

	gotPath, gotBinaries, err := installToFirstWriteableDir(paths, r, binaries)

	require.NoError(t, err)
	assert.Equal(t, paths[1], gotPath)
	assert.Equal(t, []string{"topo"}, gotBinaries)
}
