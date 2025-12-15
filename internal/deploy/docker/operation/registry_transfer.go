package operation

import (
	"fmt"
	"io"
	"os/exec"
	"sort"
	"strings"

	"github.com/arm-debug/topo-cli/internal/deploy/docker/command"
	"github.com/arm-debug/topo-cli/internal/ssh"
)

type RegistryTransfer struct {
	composeFile string
	sourceHost  ssh.Host
	targetHost  ssh.Host
}

func NewRegistryTransfer(composeFile string, sourceHost, targetHost ssh.Host) *RegistryTransfer {
	return &RegistryTransfer{composeFile: composeFile, sourceHost: sourceHost, targetHost: targetHost}
}

func (r *RegistryTransfer) Description() string {
	return "Transfer via registry"
}

func (r *RegistryTransfer) Run(w io.Writer) error {
	images, err := r.getImagesFromCompose(w)
	if err != nil {
		return err
	}
	for _, image := range images {
		if err := r.transferImage(w, image); err != nil {
			return err
		}
	}
	return nil
}

func (r *RegistryTransfer) DryRun(w io.Writer) error {
	images, err := r.getImagesFromCompose(w)
	if err != nil {
		return err
	}
	for _, image := range images {
		cmds := r.buildTransferCommands(image)
		for _, cmd := range cmds {
			_, _ = fmt.Fprintf(w, "%s\n", command.String(cmd))
		}
	}
	return nil
}

func (r *RegistryTransfer) getImagesFromCompose(w io.Writer) ([]string, error) {
	cmd := command.DockerCompose(r.sourceHost, r.composeFile, "config", "--images")
	cmd.Stderr = w
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get image names: %w", err)
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for i := range lines {
		lines[i] = strings.TrimSpace(lines[i])
	}
	sort.Strings(lines)
	return lines, nil
}

func (r *RegistryTransfer) buildTransferCommands(image string) []*exec.Cmd {
	tag := fmt.Sprintf("localhost:%d/%s", ssh.RegistryPort, image)
	return []*exec.Cmd{
		command.Docker(r.sourceHost, "tag", image, tag),
		command.Docker(r.sourceHost, "push", tag),
		command.Docker(r.targetHost, "pull", tag),
		command.Docker(r.targetHost, "tag", tag, image), // Restore original image name on target
	}
}

func (r *RegistryTransfer) transferImage(w io.Writer, image string) error {
	cmds := r.buildTransferCommands(image)
	for _, cmd := range cmds {
		cmd.Stdout = w
		cmd.Stderr = w
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to execute %s: %w", strings.Join(cmd.Args, " "), err)
		}
	}
	return nil
}
