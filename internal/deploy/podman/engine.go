package podman

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/arm/topo/internal/project"
)

type EngineExecutor struct {
	socket Socket
}

func (ex EngineExecutor) Command(ctx context.Context, args ...string) *exec.Cmd {
	return Command(ctx, ex.socket, args...)
}

func (ex EngineExecutor) RunCommand(ctx context.Context, output io.Writer, args ...string) error {
	return RunCommand(ctx, output, ex.socket, args...)
}

func (ex EngineExecutor) TagImage(ctx context.Context, output io.Writer, image, tag string) error {
	return ex.RunCommand(ctx, output, "tag", image, tag)
}

func (ex EngineExecutor) PushImage(ctx context.Context, output io.Writer, tag string) (digest string, pushErr error) {
	digestFile, err := os.CreateTemp("", "topo-registry-digest-*")
	if err != nil {
		return "", fmt.Errorf("create registry digest file: %w", err)
	}
	digestFilePath := digestFile.Name()
	if err := digestFile.Close(); err != nil {
		return "", fmt.Errorf("close registry digest file: %w", err)
	}
	defer func() {
		// #nosec G703 -- digestFilePath is returned by os.CreateTemp.
		if err := os.Remove(digestFilePath); err != nil {
			pushErr = errors.Join(pushErr, fmt.Errorf("remove registry digest file: %w", err))
		}
	}()

	if err := ex.RunCommand(ctx, output, "push", "--tls-verify=false", "--digestfile", digestFilePath, tag); err != nil {
		return "", err
	}
	// #nosec G703 -- digestFilePath is returned by os.CreateTemp.
	digestBytes, err := os.ReadFile(digestFilePath)
	if err != nil {
		return "", fmt.Errorf("read registry digest file: %w", err)
	}
	digest = strings.TrimSpace(string(digestBytes))
	if digest == "" {
		return "", fmt.Errorf("registry push did not write an image digest")
	}
	return fmt.Sprintf("%s@%s", tag, digest), nil
}

func (ex EngineExecutor) PullImage(ctx context.Context, output io.Writer, image string) error {
	return ex.RunCommand(ctx, output, "pull", "--tls-verify=false", image)
}

func (ex EngineExecutor) RunComposeCommand(ctx context.Context, output io.Writer, scope project.Scope, args ...string) error {
	return RunComposeCommand(ctx, output, ex.socket, scope, args...)
}
