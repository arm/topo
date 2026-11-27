package deploy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/arm/topo/internal/output/term"
	"github.com/arm/topo/internal/project"
)

type ComposeCommandRunner interface {
	ComposeCommand(ctx context.Context, scope project.Scope, args ...string) (*exec.Cmd, error)
	RunComposeCommand(ctx context.Context, output io.Writer, scope project.Scope, args ...string) error
}

func BuildImages(ctx context.Context, progress *term.Progress, scope project.Scope, runner ComposeCommandRunner) error {
	if err := progress.Header("Build images"); err != nil {
		return err
	}
	return runner.RunComposeCommand(ctx, progress.Output(), scope, "build")
}

func PullImages(ctx context.Context, progress *term.Progress, scope project.Scope, runner ComposeCommandRunner) error {
	if err := progress.Header("Pull images"); err != nil {
		return err
	}
	services, err := project.PullableServices(scope)
	if err != nil {
		return err
	}
	if len(services) == 0 {
		return nil
	}

	args := append([]string{"pull"}, services...)
	return runner.RunComposeCommand(ctx, progress.Output(), scope, args...)
}

func StartServices(ctx context.Context, progress *term.Progress, scope project.Scope, mode RecreateMode, runner ComposeCommandRunner) error {
	if err := progress.Header("Start services"); err != nil {
		return err
	}
	args := []string{"up", "-d", "--no-build", "--pull", "never"}
	switch mode {
	case RecreateModeForce:
		args = append(args, "--force-recreate")
	case RecreateModeNone:
		args = append(args, "--no-recreate")
	}
	return runner.RunComposeCommand(ctx, progress.Output(), scope, args...)
}

func StopServices(ctx context.Context, progress *term.Progress, scope project.Scope, runner ComposeCommandRunner) error {
	if err := progress.Header("Stop services"); err != nil {
		return err
	}
	return runner.RunComposeCommand(ctx, progress.Output(), scope, "stop")
}

func LogServices(ctx context.Context, output io.Writer, scope project.Scope, runner ComposeCommandRunner, options LogsOptions) error {
	args := []string{"logs", "--tail", "100"}
	if options.Follow {
		args = append(args, "--follow")
	}
	args = append(args, "--")
	args = append(args, options.Services...)
	return runner.RunComposeCommand(ctx, output, scope, args...)
}

type RawContainer struct {
	ID      string `json:"ID"`
	Names   string `json:"Names"`
	Service string `json:"Service"`
	Image   string `json:"Image"`
	State   string `json:"State"`
	Status  string `json:"Status"`
	Ports   string `json:"Ports"`
}

func ListContainersRaw(
	ctx context.Context,
	scope project.Scope,
	runner ComposeCommandRunner,
	all bool,
) ([]RawContainer, error) {
	args := []string{"ps", "--format", "json"}
	if all {
		args = append(args, "--all")
	}
	cmd, err := runner.ComposeCommand(ctx, scope, args...)
	if err != nil {
		return nil, err
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("%s: %w: %s", strings.Join(cmd.Args, " "), err, strings.TrimSpace(stderr.String()))
	}

	raws := []RawContainer{}
	decoder := json.NewDecoder(strings.NewReader(stdout.String()))
	for {
		var raw RawContainer
		err := decoder.Decode(&raw)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		raws = append(raws, raw)
	}
	return raws, nil
}
