package deploy

import (
	"context"

	"github.com/arm/topo/internal/output/term"
	"github.com/arm/topo/internal/project"
)

func PrepareImages(ctx context.Context, progress *term.Progress, scope project.Scope, runCompose RunComposeCommandFn) error {
	if err := BuildImages(ctx, progress, scope, runCompose); err != nil {
		return err
	}
	return PullImages(ctx, progress, scope, runCompose)
}
