package docker

import (
	"context"
	"io"

	"github.com/arm/topo/internal/deploy"
	"github.com/arm/topo/internal/project"
)

func BuildImages(ctx context.Context, output io.Writer, host Host, scope project.Scope) error {
	return RunComposeCommand(ctx, output, host, scope, "build")
}

func PullImages(ctx context.Context, output io.Writer, host Host, scope project.Scope) error {
	services, err := project.PullableServices(scope)
	if err != nil {
		return err
	}
	if len(services) == 0 {
		return nil
	}

	args := append([]string{"pull"}, services...)
	return RunComposeCommand(ctx, output, host, scope, args...)
}

func StartServices(ctx context.Context, output io.Writer, host Host, scope project.Scope, mode deploy.RecreateMode) error {
	args := []string{"up", "-d", "--no-build", "--pull", "never"}
	switch mode {
	case deploy.RecreateModeForce:
		args = append(args, "--force-recreate")
	case deploy.RecreateModeNone:
		args = append(args, "--no-recreate")
	}
	return RunComposeCommand(ctx, output, host, scope, args...)
}

func StopServices(ctx context.Context, output io.Writer, host Host, scope project.Scope) error {
	return RunComposeCommand(ctx, output, host, scope, "stop")
}
