package docker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strings"

	"github.com/arm/topo/internal/project"
)

type Engine struct {
	Host Host
}

var ContainerPortConflictMessages = []string{"already in use", "already allocated"}

func (ex Engine) Command(ctx context.Context, args ...string) *exec.Cmd {
	return Command(ctx, ex.Host, args...)
}

func (ex Engine) RunCommand(ctx context.Context, output io.Writer, args ...string) error {
	return RunCommand(ctx, output, ex.Host, args...)
}

// AssertPreservesDigestOnTag checks that a new image name can resolve with the
// original digest after tagging. Docker's legacy image store adds a tag without
// a corresponding repository digest reference; the containerd image store
// always preserves the manifest descriptor.
func (ex Engine) AssertPreservesDigestOnTag(ctx context.Context) error {
	usesContainerd, err := ex.usesContainerdImageStore(ctx)
	if err != nil {
		return err
	}
	if !usesContainerd {
		return fmt.Errorf("the Docker daemon uses the legacy image store, which cannot restore the original digest-pinned reference; enable the containerd image store (https://docs.docker.com/engine/storage/containerd/) or use a tag without @sha256:... in Compose")
	}

	return nil
}

func (ex Engine) TagImage(ctx context.Context, output io.Writer, image, tag string) error {
	return ex.RunCommand(ctx, output, "tag", image, tag)
}

func (ex Engine) PushImage(ctx context.Context, output io.Writer, image string) (digest string, pushErr error) {
	pushCommand := ex.Command(ctx, "push", image)
	var pushOutput bytes.Buffer
	pushCommand.Stdout = io.MultiWriter(output, &pushOutput)
	pushCommand.Stderr = output
	if err := pushCommand.Run(); err != nil {
		return "", fmt.Errorf("failed to execute %s: %w", strings.Join(pushCommand.Args, " "), err)
	}

	digest, pushErr = ParseDigestFromPushOutput(pushOutput.String())
	if pushErr != nil {
		return "", fmt.Errorf("failed to parse digest after pushing %s: %w", image, pushErr)
	}
	return fmt.Sprintf("%s@%s", image, digest), nil
}

func (ex Engine) PullImage(ctx context.Context, output io.Writer, image string) error {
	return ex.RunCommand(ctx, output, "pull", image)
}

func (ex Engine) RunComposeCommand(ctx context.Context, output io.Writer, scope project.Scope, args ...string) error {
	return RunComposeCommand(ctx, output, ex.Host, scope, args...)
}

func (ex Engine) usesContainerdImageStore(ctx context.Context) (bool, error) {
	output, err := ex.Command(ctx, "info", "--format", `{{json .DriverStatus}}`).Output()
	if err != nil {
		return false, fmt.Errorf("failed to check Docker image store: %w", err)
	}
	var status [][]string
	if err := json.Unmarshal(output, &status); err != nil {
		return false, fmt.Errorf("failed to decode Docker driver status: %w", err)
	}
	for _, entry := range status {
		if len(entry) == 2 && entry[0] == "driver-type" && entry[1] == "io.containerd.snapshotter.v1" {
			return true, nil
		}
	}
	return false, nil
}

var digestRegexp = regexp.MustCompile(`digest: (sha256:[a-f0-9]+)`)

func ParseDigestFromPushOutput(output string) (string, error) {
	match := digestRegexp.FindStringSubmatch(output)
	if match == nil {
		return "", fmt.Errorf("no digest found in push output")
	}
	return match[1], nil
}
