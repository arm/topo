package command

import (
	"os/exec"
	"strconv"
)

func configureCancellation(cmd *exec.Cmd) {
	// Include child processes when stopping the command.
	cmd.Cancel = func() error {
		if err := exec.Command("taskkill", "/F", "/T", "/PID", strconv.Itoa(cmd.Process.Pid)).Run(); err != nil {
			return cmd.Process.Kill()
		}
		return nil
	}
}
