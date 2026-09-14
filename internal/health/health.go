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

type TargetReport struct {
	Destination  string
	IsLocalhost  bool
	Dependencies []DependencyReport
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

func generateDependencyReport(statuses []DependencyStatus) []DependencyReport {
	reports := []DependencyReport{}
	for _, status := range statuses {
		report := DependencyReport{ID: status.Dependency.ID, Name: status.Dependency.Label}
		if status.Result.Failure == nil {
			report.Status = CheckStatusOK
			report.Value = status.Result.SuccessValue
		} else {
			report.Status = checkStatusFromSeverity(status.Result.Failure.Severity)
			report.Value = status.Result.Failure.Message
			report.Fix = status.Result.Failure.Fix
		}
		reports = append(reports, report)
	}
	return reports
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
