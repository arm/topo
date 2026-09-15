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
	graph := NewDependencyGraph(DependencyGraphOptions{SkipVersionChecks: opts.SkipVersionChecks})
	evaluatedGraph := graph.Evaluate(context.Background())
	return HostReport{Dependencies: toDependencyReports(evaluatedGraph.Host)}
}

func CheckTarget(ctx context.Context, dest ssh.Destination, acceptNewHostKeys bool) TargetReport {
	graph := NewDependencyGraph(DependencyGraphOptions{Target: &dest, AcceptHostKeys: acceptNewHostKeys})
	evaluatedGraph := graph.Evaluate(ctx)
	return TargetReport{
		Destination:  dest.String(),
		IsLocalhost:  dest.IsPlainLocalhost(),
		Dependencies: toDependencyReports(evaluatedGraph.Target),
	}
}

func ToDependencyReport(status DependencyStatus) DependencyReport {
	report := DependencyReport{ID: status.ID, Name: status.Label}
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
