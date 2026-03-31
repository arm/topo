package operation

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"sort"
	"strings"

	"github.com/arm/topo/internal/deploy/docker/command"
)

var digestRegexp = regexp.MustCompile(`digest: (sha256:[a-f0-9]+)`)

type RegistryTransfer struct {
	composeFile string
	source      command.Host
	host        command.Host
	port        string
}

func NewRegistryTransfer(composeFile string, sourceHost, dest command.Host, port string) *RegistryTransfer {
	return &RegistryTransfer{composeFile: composeFile, source: sourceHost, host: dest, port: port}
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

func (r *RegistryTransfer) getImagesFromCompose(w io.Writer) ([]string, error) {
	cmd := command.DockerCompose(r.source, r.composeFile, "config", "--images")
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

func (r *RegistryTransfer) transferImage(w io.Writer, image string) error {
	tag := fmt.Sprintf("localhost:%s/%s", r.port, image)

	tagCmd := command.Docker(r.source, "tag", image, tag)
	if err := runCmd(tagCmd, w); err != nil {
		return err
	}

	pushCmd := command.Docker(r.source, "push", tag)
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
	pullCmd := command.Docker(r.host, "pull", digestRef)
	if err := runCmd(pullCmd, w); err != nil {
		return err
	}

	retagCmd := command.Docker(r.host, "tag", digestRef, image)
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
	cmd := command.Docker(command.LocalHost, "port", RegistryContainerName, "5000")
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
