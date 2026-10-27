package command

import (
	"os/exec"
	"time"
)

// ConfigureCancellation extends exec.CommandContext cancellation to child processes.
// It must be called before starting the command.
func ConfigureCancellation(cmd *exec.Cmd) {
	cmd.WaitDelay = 5 * time.Second
	configureCancellation(cmd)
}
