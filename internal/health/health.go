package health

import (
	"context"

	"github.com/arm/topo/internal/ssh"
)

type CheckStatus string

const (
	CheckStatusOK      CheckStatus = "ok"
	CheckStatusWarning CheckStatus = "warning"
	CheckStatusError   CheckStatus = "error"
	CheckStatusInfo    CheckStatus = "info"
)

type HealthCheck struct {
	ID     DependencyID
	Name   string
	Status CheckStatus
	Value  string
	Fix    *Fix
}

type HostReport struct {
	Dependencies []HealthCheck
}

type TargetReport struct {
	Destination  string
	IsLocalhost  bool
	Dependencies []HealthCheck
}

type CheckHostOptions struct {
	SkipVersionChecks bool
}

func CheckHost(opts CheckHostOptions) HostReport {
	deps := HostRequiredDependencies(opts.SkipVersionChecks)
	dependencyStatuses := PerformChecks(context.Background(), deps)
	return GenerateHostReport(dependencyStatuses)
}

type Status struct {
	Destination  ssh.Destination
	Dependencies []DependencyStatus
}

func CheckTarget(ctx context.Context, dest ssh.Destination, acceptNewHostKeys bool) TargetReport {
	targetDependencyStatuses := PerformChecks(ctx, TargetRequiredDependencies(dest, acceptNewHostKeys))
	return GenerateTargetReport(Status{Destination: dest, Dependencies: targetDependencyStatuses})
}

func GenerateHostReport(statuses []DependencyStatus) HostReport {
	report := HostReport{}
	report.Dependencies = generateDependencyReport(statuses)

	return report
}

func GenerateTargetReport(targetStatus Status) TargetReport {
	return TargetReport{
		Destination:  targetStatus.Destination.String(),
		IsLocalhost:  targetStatus.Destination.IsPlainLocalhost(),
		Dependencies: generateDependencyReport(targetStatus.Dependencies),
	}
}

func generateDependencyReport(statuses []DependencyStatus) []HealthCheck {
	res := []HealthCheck{}
	for _, ds := range statuses {
		hc := HealthCheck{ID: ds.Dependency.ID, Name: ds.Dependency.Label}
		if ds.Result.Failure == nil {
			hc.Status = CheckStatusOK
			hc.Value = ds.Result.SuccessValue
		} else {
			hc.Status = checkStatusFromSeverity(ds.Result.Failure.Severity)
			hc.Value = ds.Result.Failure.Message
			hc.Fix = ds.Result.Failure.Fix
		}
		res = append(res, hc)
	}
	return res
}

func checkStatusFromSeverity(severity CheckSeverity) CheckStatus {
	switch severity {
	case SeverityWarning:
		return CheckStatusWarning
	case SeverityInfo:
		return CheckStatusInfo
	default:
		return CheckStatusError
	}
}
