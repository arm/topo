package ssh

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/arm/topo/internal/command"
)

func ControlSocketPath(targetHost string) string {
	hash := sha256.Sum256([]byte(targetHost))
	hostHash := fmt.Sprintf("%x", hash[:8]) // Hash to avoid filepath limits
	return filepath.Join(os.TempDir(), fmt.Sprintf("topo-tunnel-%s", hostHash))
}

type Tunnel struct {
	useControlSockets bool
	dest              string

	command *exec.Cmd
	closed  bool
}

func OpenTunnel(ctx context.Context, w io.Writer, dest Destination, port string) (*Tunnel, error) {
	t := &Tunnel{
		useControlSockets: areControlSocketsSupported(),
		dest:              dest.String(),
	}

	args := []string{"ssh", "-N", "-o", "ExitOnForwardFailure=yes"}
	if t.useControlSockets {
		args = append(args,
			"-fMS", ControlSocketPath(t.dest),
		)
	}
	args = append(
		args,
		"-R", fmt.Sprintf("127.0.0.1:%s:127.0.0.1:%s", port, port),
		t.dest,
	)
	// #nosec -- arguments are validated
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Stdout = w
	cmd.Stderr = w

	if t.useControlSockets {
		if err := cmd.Run(); err != nil {
			return nil, command.FormatError(cmd.Args, err)
		}
	} else {
		if err := cmd.Start(); err != nil {
			return nil, command.FormatError(cmd.Args, err)
		}
		t.command = cmd
	}

	return t, nil
}

func (t *Tunnel) Close(ctx context.Context, w io.Writer) error {
	if t.closed {
		return nil
	}

	var err error
	if t.useControlSockets {
		args := []string{
			"ssh",
			"-S", ControlSocketPath(t.dest),
			"-O", "exit",
			t.dest,
		}
		// #nosec -- arguments are validated
		cmd := exec.CommandContext(ctx, args[0], args[1:]...)
		cmd.Stdout = w
		cmd.Stderr = w
		if runErr := cmd.Run(); runErr != nil {
			err = command.FormatError(cmd.Args, runErr)
		}
	} else {
		err = killCommand(t.command)
	}
	if err != nil {
		return err
	}

	t.closed = true
	return nil
}

func areControlSocketsSupported() bool {
	return runtime.GOOS != "windows"
}

func killCommand(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}

	pid := cmd.Process.Pid
	if err := cmd.Process.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
		return fmt.Errorf("kill process %d: %w", pid, err)
	}

	if err := cmd.Wait(); err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) && !errors.Is(err, os.ErrProcessDone) {
			return fmt.Errorf("wait for process %d: %w", pid, err)
		}
	}
	return nil
}
