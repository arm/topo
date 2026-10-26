//go:build !windows

package command

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
	"time"
)

func configureCancellation(cmd *exec.Cmd) {
	// Interrupt the provider as well as the engine, like terminal Ctrl+C.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		return cancelProcessGroup(cmd.Process.Pid)
	}
}

func cancelProcessGroup(pid int) error {
	if err := signalProcessGroup(pid, syscall.SIGINT); err != nil {
		return err
	}

	// The wrapper may exit before its children. Wait for the group, not just
	// the parent process, before deciding whether forced cleanup is needed.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if err := signalProcessGroup(pid, 0); err != nil {
			if errors.Is(err, os.ErrProcessDone) {
				return nil
			}
			return err
		}
		time.Sleep(25 * time.Millisecond)
	}
	err := signalProcessGroup(pid, syscall.SIGKILL)
	if errors.Is(err, os.ErrProcessDone) {
		return nil
	}
	return err
}

func signalProcessGroup(pid int, signal syscall.Signal) error {
	err := syscall.Kill(-pid, signal)
	if errors.Is(err, syscall.ESRCH) {
		return os.ErrProcessDone
	}
	return err
}
