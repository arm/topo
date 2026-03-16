package e2e

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnknownCommand(t *testing.T) {
	bin := buildBinary(t)

	cmd := exec.Command(bin, "yolo-swag")
	out, err := cmd.CombinedOutput()

	assert.Error(t, err)
	want := `unknown command "yolo-swag"`
	assert.Contains(t, string(out), want)
}
