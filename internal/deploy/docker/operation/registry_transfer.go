package operation

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"sort"
	"strings"

	"github.com/arm/topo/internal/command"
	"github.com/arm/topo/internal/ssh"
)

var digestRegexp = regexp.MustCompile(`digest: (sha256:[a-f0-9]+)`)

type RegistryTransfer struct {
	composeFile string
	sourceHost  ssh.Destination
	dest  ssh.Destination
	port        string
}

func NewRegistryTransfer(composeFile string, sourceHost, dest ssh.Destination, port string) *RegistryTransfer {
	return &RegistryTransfer{composeFile: composeFile, sourceHost: sourceHost, dest: dest, port: port}
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
	tag := fmt.Sprintf("localhost:%s/%s", r.port, image)
	digestRef := fmt.Sprintf("localhost:%s/%s@<digest>", r.port, image)
	return []*exec.Cmd{
		command.Docker(r.sourceHost, "tag", image, tag),
		command.Docker(r.sourceHost, "push", tag),
		command.Docker(r.dest, "pull", digestRef),
		command.Docker(r.dest, "tag", digestRef, image),
	}
}

func (r *RegistryTransfer) transferImage(w io.Writer, image string) error {
	tag := fmt.Sprintf("localhost:%s/%s", r.port, image)

	tagCmd := command.Docker(r.sourceHost, "tag", image, tag)
	if err := runCmd(tagCmd, w); err != nil {
		return err
	}

	pushCmd := command.Docker(r.sourceHost, "push", tag)
	pushOutput, err := runCmdCaptureOutput(pushCmd, w)
	if err != nil {
		if hint := r.checkRegistryPortMismatch(); hint != "" {
			return fmt.Errorf("%s\n%s", err, hint)
		}
		return fmt.Errorf("failed to execute %s: %w", strings.Join(pushCmd.Args, " "), err)
	}

	digest, err := ParseDigestFromPushOutput(pushOutput)
	if err != nil {
		return fmt.Errorf("failed to parse digest after pushing %s: %w", tag, err)
	}

	digestRef := fmt.Sprintf("localhost:%s/%s@%s", r.port, image, digest)
	pullCmd := command.Docker(r.dest, "pull", digestRef)
	if err := runCmd(pullCmd, w); err != nil {
		return err
	}

	retagCmd := command.Docker(r.dest, "tag", digestRef, image)
	if err := runCmd(retagCmd, w); err != nil {
		return err
	}

	return nil
}

func runCmd(cmd *exec.Cmd, w io.Writer) error {
	cmd.Stdout = w
	cmd.Stderr = w
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to execute %s: %w", strings.Join(cmd.Args, " "), err)
	}
	return nil
}

func runCmdCaptureOutput(cmd *exec.Cmd, w io.Writer) (string, error) {
	var buf bytes.Buffer
	cmd.Stdout = io.MultiWriter(w, &buf)
	cmd.Stderr = w
	err := cmd.Run()
	return buf.String(), err
}

func ParseDigestFromPushOutput(output string) (string, error) {
	match := digestRegexp.FindStringSubmatch(output)
	if match == nil {
		return "", fmt.Errorf("no digest found in push output")
	}
	return match[1], nil
}

func (r *RegistryTransfer) checkRegistryPortMismatch() string {
	cmd := command.Docker(ssh.PlainLocalhost, "port", RegistryContainerName, "5000")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	actual := strings.TrimSpace(string(out))
	if idx := strings.LastIndex(actual, ":"); idx != -1 {
		actualPort := actual[idx+1:]
		if actualPort != r.port {
			return fmt.Sprintf("ERROR: Registry port mismatch (running: %s, requested: %s)\nYou may need to stop the existing topo-registry: docker rm -f %s", actualPort, r.port, RegistryContainerName)
		}
	}
	return ""
}
