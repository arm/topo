package health_test

import (
	"context"

	"github.com/arm/topo/internal/health"
)

func passingCheck(_ context.Context) health.DependencyCheckResult {
	return health.DependencyCheckResult{SuccessValue: "passed"}
}

func failingCheck(_ context.Context) health.DependencyCheckResult {
	return health.DependencyCheckResult{Failure: &health.DependencyCheckFailure{
		Severity: health.SeverityError,
		Message:  "very broken",
		Fix:      &health.Fix{Description: "fix me please", Command: "echo fixed"},
	}}
}
