package observations

import (
	"context"
	"strings"
)

// IdentityPrefixer adds the provider identity and target scope to local observation trees on the host.
type IdentityPrefixer struct {
	identity string
	provider Provider
}

func NewIdentityPrefixer(identity string, provider Provider) *IdentityPrefixer {
	return &IdentityPrefixer{identity: identity, provider: provider}
}

func (c *IdentityPrefixer) ObserveTarget(ctx context.Context, runner TargetRunner) (Tree, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	local, err := c.provider.ObserveTarget(ctx, runner)
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if err != nil {
		return nil, err
	}
	tree := Tree{"target": local}
	segments := strings.Split(c.identity, ".")
	for i := len(segments) - 1; i >= 0; i-- {
		tree = Tree{segments[i]: tree}
	}
	return tree, nil
}
