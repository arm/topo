package operation

import (
	"fmt"
	"io"
	"os/exec"

	"github.com/arm-debug/topo-cli/internal/deploy/docker/command"
	"github.com/arm-debug/topo-cli/internal/ssh"
)

// Container engines on non-Linux hosts use Linux virtual machines to run containers,
// creating network isolation between the VM and host.
// VMHostRegistryBridge routes registry traffic across this boundary while preserving
// localhost addressing to prevent the registry client demanding SSL.
// The SSH tunnel is encrypted in transit so this does not downgrade security.
type VMHostRegistryBridge struct{}

func NewVMHostRegistryBridge() *VMHostRegistryBridge {
	return &VMHostRegistryBridge{}
}

const (
	forwardingContainerName = "topo-registry-routing"
	forwardingImage         = "alpine/socat:1.8.0.3"
)

func socatListenAddr() string {
	return fmt.Sprintf("TCP-LISTEN:%d,fork,reuseaddr", ssh.RegistryPort)
}

func socatConnectAddr() string {
	return fmt.Sprintf("TCP:host.docker.internal:%d", ssh.RegistryPort)
}

func (e *VMHostRegistryBridge) Description() string {
	return "Ensure TCP forwarding container exists"
}

func (e *VMHostRegistryBridge) Run(w io.Writer) error {
	pullCmd := command.Docker(ssh.PlainLocalhost, "pull", forwardingImage)
	pullCmd.Stdout = w
	pullCmd.Stderr = w
	if err := pullCmd.Run(); err != nil {
		return fmt.Errorf("failed to pull forwarding image: %w", err)
	}

	cmd := e.buildCommand()
	cmd.Stdout = w
	cmd.Stderr = w
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to ensure forwarding container: %w", err)
	}
	return nil
}

func (e *VMHostRegistryBridge) DryRun(w io.Writer) error {
	pullCmd := command.Docker(ssh.PlainLocalhost, "pull", forwardingImage)
	_, _ = fmt.Fprintln(w, command.String(pullCmd))
	_, _ = fmt.Fprintln(w, command.String(e.buildRunCommand()))
	return nil
}

func (e *VMHostRegistryBridge) buildCommand() *exec.Cmd {
	if e.containerExists() {
		return e.buildStartCommand()
	}
	return e.buildRunCommand()
}

func (e *VMHostRegistryBridge) containerExists() bool {
	cmd := command.Docker(ssh.PlainLocalhost, "inspect", forwardingContainerName)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	return cmd.Run() == nil
}

func (e *VMHostRegistryBridge) buildStartCommand() *exec.Cmd {
	return command.Docker(ssh.PlainLocalhost, "start", forwardingContainerName)
}

func (e *VMHostRegistryBridge) buildRunCommand() *exec.Cmd {
	return command.Docker(ssh.PlainLocalhost,
		"run", "-d", "--restart=unless-stopped",
		fmt.Sprintf("--name=%s", forwardingContainerName),
		"--network=host",
		forwardingImage,
		"-ly", socatListenAddr(), socatConnectAddr(),
	)
}
