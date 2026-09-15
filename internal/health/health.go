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

type DependencyReport struct {
	ID     DependencyID
	Name   string
	Status CheckStatus
	Value  string
	Fix    *Fix
}

type HostReport struct {
	Dependencies []DependencyReport
}

type CheckHostOptions struct {
	SkipVersionChecks bool
}

type TargetReport struct {
	Destination  string
	IsLocalhost  bool
	Dependencies []DependencyReport
}

func CheckHost(opts CheckHostOptions) HostReport {
	deps := HostRequiredDependencies(opts.SkipVersionChecks)
	dependencyStatuses := PerformChecks(context.Background(), deps)
	return HostReport{
		Dependencies: toDependencyReports(dependencyStatuses),
	}
}

func CheckTarget(ctx context.Context, dest ssh.Destination, acceptNewHostKeys bool) TargetReport {
	targetDependencyStatuses := PerformChecks(ctx, TargetRequiredDependencies(dest, acceptNewHostKeys))
	return TargetReport{
		Destination:  dest.String(),
		IsLocalhost:  dest.IsPlainLocalhost(),
		Dependencies: toDependencyReports(targetDependencyStatuses),
	}
}

func ToDependencyReport(status DependencyStatus) DependencyReport {
	report := DependencyReport{ID: status.Dependency.ID, Name: status.Dependency.Label}
	if status.Result.Failure == nil {
		report.Status = CheckStatusOK
		report.Value = status.Result.SuccessValue
		return report
	}

	report.Status = checkStatusFromSeverity(status.Result.Failure.Severity)
	report.Value = status.Result.Failure.Message
	report.Fix = status.Result.Failure.Fix
	return report
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

func toDependencyReports(statuses []DependencyStatus) []DependencyReport {
	reports := make([]DependencyReport, len(statuses))
	for i, status := range statuses {
		reports[i] = ToDependencyReport(status)
	}
	return reports
}
