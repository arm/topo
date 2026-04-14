package command

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
)

type lockedWriter struct {
	mu sync.Mutex
	w  io.Writer
}

func (lw *lockedWriter) Write(p []byte) (int, error) {
	lw.mu.Lock()
	defer lw.mu.Unlock()
	return lw.w.Write(p)
}

func execute(cmd *exec.Cmd, run func() error, w io.Writer) error {
	lw := &lockedWriter{w: w}
	var stderrBuf bytes.Buffer
	cmd.Stdout = lw
	cmd.Stderr = io.MultiWriter(lw, &stderrBuf)
	if err := run(); err != nil {
		return newExecutionError(cmd, &stderrBuf, err)
	}
	return nil
}

func StartCommand(cmd *exec.Cmd, w io.Writer) error {
	return execute(cmd, cmd.Start, w)
}

func RunCommand(cmd *exec.Cmd, w io.Writer) error {
	return execute(cmd, cmd.Run, w)
}

func newExecutionError(cmd *exec.Cmd, stderr *bytes.Buffer, err error) error {
	cmdStr := strings.Join(cmd.Args, " ")
	return fmt.Errorf("%s failed: %w | stderr: %s", cmdStr, err, stderr.String())
}
