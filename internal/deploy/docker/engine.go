package docker

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strings"

	"github.com/arm/topo/internal/project"
)

type EngineExecutor struct {
	host Host
}

func (ex EngineExecutor) Command(ctx context.Context, args ...string) *exec.Cmd {
	return Command(ctx, ex.host, args...)
}

func (ex EngineExecutor) RunCommand(ctx context.Context, output io.Writer, args ...string) error {
	return RunCommand(ctx, output, ex.host, args...)
}

func (ex EngineExecutor) TagImage(ctx context.Context, output io.Writer, image, tag string) error {
	return ex.RunCommand(ctx, output, "tag", image, tag)
}

func (ex EngineExecutor) PushImage(ctx context.Context, output io.Writer, image string) (digest string, pushErr error) {
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

func (ex EngineExecutor) PullImage(ctx context.Context, output io.Writer, image string) error {
	return ex.RunCommand(ctx, output, "pull", image)
}

func (ex EngineExecutor) RunComposeCommand(ctx context.Context, output io.Writer, scope project.Scope, args ...string) error {
	return RunComposeCommand(ctx, output, ex.host, scope, args...)
}

var digestRegexp = regexp.MustCompile(`digest: (sha256:[a-f0-9]+)`)

func ParseDigestFromPushOutput(output string) (string, error) {
	match := digestRegexp.FindStringSubmatch(output)
	if match == nil {
		return "", fmt.Errorf("no digest found in push output")
	}
	return match[1], nil
}
