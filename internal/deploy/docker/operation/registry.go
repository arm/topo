package operation

import (
	"fmt"
	"io"
	"os/exec"

	"github.com/arm-debug/topo-cli/internal/deploy/docker/command"
	"github.com/arm-debug/topo-cli/internal/deploy/operation"
	"github.com/arm-debug/topo-cli/internal/ssh"
	"golang.org/x/sync/errgroup"
)

const (
	RegistryContainerName = "topo-registry"
	registryImage         = "registry:2"
)

func NewRunRegistry() operation.Sequence {
	return operation.NewSequence(
		NewDockerPull(ssh.PlainLocalhost, registryImage),
		NewStartOrRun(ssh.PlainLocalhost, RegistryContainerName, registryImage,
			"-d", "--restart", "always", fmt.Sprintf("-p=127.0.0.1:%d:5000", ssh.RegistryPort)),
	)
}

type PipeTransfer struct {
	image      string
	sourceHost ssh.Host
	targetHost ssh.Host
}

func NewPipeTransfer(image string, sourceHost, targetHost ssh.Host) *PipeTransfer {
	return &PipeTransfer{image: image, sourceHost: sourceHost, targetHost: targetHost}
}

func (t *PipeTransfer) Description() string {
	return fmt.Sprintf("Transfer image %s", t.image)
}

func (t *PipeTransfer) Run(w io.Writer) error {
	saveCmd := command.Docker(t.sourceHost, "save", t.image)
	loadCmd := command.Docker(t.targetHost, "load")
	return t.pipe(w, saveCmd, loadCmd)
}

func (t *PipeTransfer) DryRun(w io.Writer) error {
	saveCmd := command.Docker(t.sourceHost, "save", t.image)
	loadCmd := command.Docker(t.targetHost, "load")
	_, err := fmt.Fprintf(w, "%s | %s\n", command.String(saveCmd), command.String(loadCmd))
	return err
}

func (t *PipeTransfer) pipe(w io.Writer, saveCmd, loadCmd *exec.Cmd) error {
	pipeReader, pipeWriter := io.Pipe()
	saveCmd.Stdout = pipeWriter
	saveCmd.Stderr = w
	loadCmd.Stdin = pipeReader
	loadCmd.Stdout = w
	loadCmd.Stderr = w

	var g errgroup.Group
	g.Go(func() error {
		defer pipeWriter.Close() //nolint:errcheck
		if err := saveCmd.Run(); err != nil {
			return fmt.Errorf("failed to save image: %w", err)
		}
		return nil
	})
	g.Go(func() error {
		defer pipeReader.Close() //nolint:errcheck
		if err := loadCmd.Run(); err != nil {
			return fmt.Errorf("failed to load image: %w", err)
		}
		return nil
	})
	return g.Wait()
}

type StartOrRun struct {
	host          ssh.Host
	containerName string
	image         string
	runArgs       []string
}

func NewStartOrRun(host ssh.Host, containerName, image string, runArgs ...string) *StartOrRun {
	return &StartOrRun{host: host, containerName: containerName, image: image, runArgs: runArgs}
}

func (s *StartOrRun) Description() string {
	return s.buildOperation().Description()
}

func (s *StartOrRun) Run(w io.Writer) error {
	return s.buildOperation().Run(w)
}

func (s *StartOrRun) DryRun(w io.Writer) error {
	return s.buildOperation().DryRun(w)
}

func (s *StartOrRun) buildOperation() operation.Operation {
	if s.containerExists() {
		return NewDockerStart(s.host, s.containerName)
	}
	return NewDockerRun(s.host, s.image, s.containerName, s.runArgs)
}

func (s *StartOrRun) containerExists() bool {
	cmd := command.Docker(s.host, "inspect", s.containerName)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	return cmd.Run() == nil
}
