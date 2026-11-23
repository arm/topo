// Package observations defines target observations supplied through Topo-owned target access.
package observations

import "context"

type Provider interface {
	ObserveTarget(context.Context, TargetRunner) (Tree, error)
}

type TargetRunner interface {
	Run(context.Context, string) (stdout, stderr string, err error)
}
