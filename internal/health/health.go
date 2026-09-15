package health

import "context"

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

type HealthReport struct {
	Host   HostReport
	Target *TargetReport
}

func Check(ctx context.Context, options DependencyGraphOptions) HealthReport {
	graph := NewDependencyGraph(options)
	evaluatedGraph := graph.Evaluate(ctx)
	report := HealthReport{
		Host: HostReport{Dependencies: toDependencyReports(evaluatedGraph.Host)},
	}
	if graph.Target == nil {
		return report
	}

	targetReport := TargetReport{
		Destination:  options.Target.String(),
		IsLocalhost:  options.Target.IsPlainLocalhost(),
		Dependencies: toDependencyReports(evaluatedGraph.Target),
	}
	report.Target = &targetReport
	return report
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
