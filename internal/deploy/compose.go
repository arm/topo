package deploy

import (
	"context"
	"io"

	"github.com/arm/topo/internal/output/term"
	"github.com/arm/topo/internal/project"
)

type ComposeCommandRunner interface {
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
