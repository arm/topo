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

func NewRunRegistry(host ssh.Host) operation.Sequence {
	return operation.NewSequence(
		NewPull(ssh.PlainLocalhost, registryImage),
		NewPipeTransfer(registryImage, ssh.PlainLocalhost, host),
		NewStartOrRun(host, RegistryContainerName, registryImage,
			"-d", "--restart=always", fmt.Sprintf("-p=127.0.0.1:%d:5000", ssh.RegistryPort)),
	)
}

type Pull struct {
	host  ssh.Host
	image string
}

func NewPull(host ssh.Host, image string) *Pull {
	return &Pull{host: host, image: image}
}

func (p *Pull) Description() string {
	return fmt.Sprintf("Pull image %s", p.image)
}

func (p *Pull) Run(w io.Writer) error {
	cmd := command.Docker(p.host, "pull", p.image)
	cmd.Stdout = w
	cmd.Stderr = w
	return cmd.Run()
}

func (p *Pull) DryRun(w io.Writer) error {
	cmd := command.Docker(p.host, "pull", p.image)
	_, err := fmt.Fprintln(w, command.String(cmd))
	return err
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
	return fmt.Sprintf("Start image %s", s.containerName)
}

func (s *StartOrRun) Run(w io.Writer) error {
	cmd := s.buildCommand()
	cmd.Stdout = w
	cmd.Stderr = w
	return cmd.Run()
}

func (s *StartOrRun) DryRun(w io.Writer) error {
	cmd := s.buildCommand()
	_, err := fmt.Fprintln(w, command.String(cmd))
	return err
}

func (s *StartOrRun) buildCommand() *exec.Cmd {
	if s.containerExists() {
		return s.buildStartCommand()
	}
	return s.buildRunCommand()
}

func (s *StartOrRun) containerExists() bool {
	cmd := command.Docker(s.host, "inspect", s.containerName)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	return cmd.Run() == nil
}

func (s *StartOrRun) buildStartCommand() *exec.Cmd {
	return command.Docker(s.host, "start", s.containerName)
}

func (s *StartOrRun) buildRunCommand() *exec.Cmd {
	args := append([]string{"run"}, s.runArgs...)
	args = append(args, fmt.Sprintf("--name=%s", s.containerName), s.image)
	return command.Docker(s.host, args...)
}
