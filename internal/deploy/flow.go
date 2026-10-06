package deploy

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	"github.com/arm/topo/internal/output/term"
	"github.com/arm/topo/internal/project"
	"github.com/arm/topo/internal/ssh"
	"golang.org/x/sync/errgroup"
)

func PrepareImages(ctx context.Context, progress *term.Progress, scope project.Scope, runCompose RunComposeCommandFn) error {
	if err := BuildImages(ctx, progress, scope, runCompose); err != nil {
		return err
	}
	return PullImages(ctx, progress, scope, runCompose)
}

type RunSaveCommandFn func(ctx context.Context, output io.Writer, image string, imagePayload io.Writer) error

type RunLoadCommandFn func(ctx context.Context, output io.Writer, imagePayload io.Reader) error

func TransferImagesViaPipe(
	ctx context.Context,
	progress *term.Progress,
	runSave RunSaveCommandFn,
	runLoad RunLoadCommandFn,
	scope project.Scope,
) error {
	if err := progress.Header("Transfer images"); err != nil {
		return err
	}

	images, err := project.ImageNames(scope)
	if err != nil {
		return err
	}

	var group errgroup.Group
	for _, image := range images {
		group.Go(func() error {
			return transferImageViaPipe(ctx, progress.Output(), runSave, runLoad, image)
		})
	}
	return group.Wait()
}

func transferImageViaPipe(
	ctx context.Context,
	output io.Writer,
	runSave RunSaveCommandFn,
	runLoad RunLoadCommandFn,
	image string,
) error {
	pipeReader, pipeWriter := io.Pipe()

	var group errgroup.Group
	group.Go(func() error {
		err := runSave(ctx, output, image, pipeWriter)
		_ = pipeWriter.CloseWithError(err)
		if err != nil {
			return fmt.Errorf("failed to save image %s: %w", image, err)
		}
		return nil
	})
	group.Go(func() error {
		err := runLoad(ctx, output, pipeReader)
		_ = pipeReader.CloseWithError(err)
		if err != nil {
			return fmt.Errorf("failed to load image %s: %w", image, err)
		}
		return nil
	})
	return group.Wait()
}

type EngineExecutor interface {
	Command(ctx context.Context, args ...string) *exec.Cmd
	RunCommand(ctx context.Context, output io.Writer, args ...string) error
}

func PrepareRegistry(
	ctx context.Context,
	progress *term.Progress,
	config RegistryConfig,
	ex EngineExecutor,
	knownErrors []string,
) error {
	if err := progress.Header("Run registry"); err != nil {
		return err
	}

	registryContainerExists := ex.RunCommand(ctx, io.Discard, "inspect", config.ContainerName) == nil
	if registryContainerExists {
		if err := validateRegistryPort(ctx, ex, config.ContainerName, config.Port); err != nil {
			return err
		}
		return ex.RunCommand(ctx, progress.Output(), "start", config.ContainerName)
	}

	var commandOutput bytes.Buffer
	combinedOutput := io.MultiWriter(progress.Output(), &commandOutput)
	err := ex.RunCommand(
		ctx,
		combinedOutput,
		"run",
		"--pull=missing",
		"-d",
		"--restart", "always",
		"-p", fmt.Sprintf("127.0.0.1:%s:5000", config.Port),
		"--name", config.ContainerName,
		"registry:2",
	)
	if err == nil {
		return nil
	}
	commandErrorOutput := strings.ToLower(commandOutput.String())
	for _, knownError := range knownErrors {
		if strings.Contains(commandErrorOutput, knownError) {
			return fmt.Errorf("%w\nport is already in use, this could be an existing %s or another process", err, config.ContainerName)
		}
	}
	return err
}

func validateRegistryPort(ctx context.Context, ex EngineExecutor, containerName, requestedPort string) error {
	cmd := ex.Command(ctx, "inspect", containerName)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to execute %s: %w", strings.Join(cmd.Args, " "), err)
	}
	actualPort, err := registryHostPort(output)
	if err != nil {
		return fmt.Errorf("failed to inspect existing registry %s: %w", containerName, err)
	}
	if actualPort == requestedPort {
		return nil
	}
	return fmt.Errorf(
		"registry port mismatch (running: %s, requested: %s)\nyou may need to remove the existing registry container %s",
		actualPort, requestedPort, containerName,
	)
}

func registryHostPort(inspectOutput []byte) (string, error) {
	type portBinding struct {
		HostPort string `json:"HostPort"`
	}
	type containerInspect struct {
		State struct {
			Running bool `json:"Running"`
		} `json:"State"`
		HostConfig struct {
			PortBindings map[string][]portBinding `json:"PortBindings"`
		} `json:"HostConfig"`
		NetworkSettings struct {
			Ports map[string][]portBinding `json:"Ports"`
		} `json:"NetworkSettings"`
	}

	var containers []containerInspect
	if err := json.Unmarshal(inspectOutput, &containers); err != nil {
		return "", fmt.Errorf("decode registry inspect output: %w", err)
	}
	if len(containers) != 1 {
		return "", fmt.Errorf("expected one inspected container, got %d", len(containers))
	}

	bindings := containers[0].HostConfig.PortBindings["5000/tcp"]
	if containers[0].State.Running {
		bindings = containers[0].NetworkSettings.Ports["5000/tcp"]
	}
	if len(bindings) == 0 || bindings[0].HostPort == "" {
		return "", fmt.Errorf("container port 5000 is not published")
	}
	return bindings[0].HostPort, nil
}

type tunnelCloseFn = func(*term.Progress) error

func OpenRegistrySSHTunnel(ctx context.Context, progress *term.Progress, target ssh.Destination, config RegistryConfig) (tunnelCloseFn, error) {
	if err := progress.Header("Open registry SSH tunnel"); err != nil {
		return nil, err
	}
	output := progress.Output()
	tunnel, err := ssh.OpenTunnel(ctx, output, target, config.Port)
	if err != nil {
		return nil, fmt.Errorf("failed to open SSH tunnel: %w; ensure port %s is free or specify a different one with `--registry-port`", err, config.Port)
	}

	close := func(progress *term.Progress) error {
		return closeTunnel(progress, tunnel)
	}

	if !target.IsLocalhost() && !config.SkipRemotePortCheck {
		if err := progress.Header("Check registry tunnel is not exposed on remote network"); err != nil {
			return nil, errors.Join(err, close(progress))
		}
		if err := CheckTunnelExposure(ctx, output, target, config.Port); err != nil {
			return nil, errors.Join(err, close(progress))
		}
	}
	return close, nil
}

func closeTunnel(progress *term.Progress, tunnel *ssh.Tunnel) error {
	const tunnelCleanupTimeout = 5 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), tunnelCleanupTimeout)
	defer cancel()

	output := progress.Output()
	var headerError error
	if output != nil {
		headerError = progress.Header("Close registry SSH tunnel")
	}
	closeError := tunnel.Close(ctx, output)
	if closeError != nil {
		closeError = fmt.Errorf("failed to close SSH tunnel: %w", closeError)
	}
	return errors.Join(headerError, closeError)
}

type RegistryEngineExecutor interface {
	TagImage(ctx context.Context, output io.Writer, image, tag string) error
	PushImage(ctx context.Context, output io.Writer, image string) (digest string, err error)
	PullImage(ctx context.Context, output io.Writer, digest string) error
}

func TransferImagesViaRegistry(
	ctx context.Context,
	progress *term.Progress,
	scope project.Scope,
	localRegistryPort string,
	sourceEx RegistryEngineExecutor,
	targetEx RegistryEngineExecutor,
) error {
	if err := progress.Header("Transfer via registry"); err != nil {
		return err
	}

	images, err := project.ImageNames(scope)
	if err != nil {
		return err
	}

	for _, image := range images {
		registryTag := fmt.Sprintf("localhost:%s/%s", localRegistryPort, image)
		if err := sourceEx.TagImage(ctx, progress.Output(), image, registryTag); err != nil {
			return err
		}
		digest, err := sourceEx.PushImage(ctx, progress.Output(), registryTag)
		if err != nil {
			return err
		}
		if err := targetEx.PullImage(ctx, progress.Output(), digest); err != nil {
			return err
		}
		if err := targetEx.TagImage(ctx, progress.Output(), digest, image); err != nil {
			return err
		}
	}

	return nil
}
