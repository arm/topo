package e2e

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListTemplates(t *testing.T) {
	bin := buildBinary(t)

	cmd := exec.Command(bin, "templates")
	out, err := cmd.CombinedOutput()
	require.NoError(t, err)

	output := string(out)

	assert.Contains(t, output, "Hello-World")
	assert.Contains(t, output, "git@github.com:")
	assert.Contains(t, output, "Features:")
}

func TestUnknownCommand(t *testing.T) {
	bin := buildBinary(t)

	cmd := exec.Command(bin, "yolo-swag")
	out, err := cmd.CombinedOutput()

	assert.Error(t, err)
	want := `unknown command "yolo-swag"`
	assert.Contains(t, string(out), want)
}
