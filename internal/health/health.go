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

type TargetDetails struct {
	Destination string
	IsLocalhost bool
}

type ReadinessReport struct {
	Host   []DependencyReport
	Target []DependencyReport
}

type HealthReport struct {
	TargetDetails    TargetDetails
	Deployment       ReadinessReport
	ProjectDiscovery ReadinessReport
}

func Check(ctx context.Context, options HealthCheckOptions) HealthReport {
	healthCheck := NewHealthCheck(options)
	evaluatedHealthCheck := healthCheck.Evaluate(ctx)
	return HealthReport{
		TargetDetails:    targetDetails(options),
		Deployment:       toReadinessReport(evaluatedHealthCheck.Deployment),
		ProjectDiscovery: toReadinessReport(evaluatedHealthCheck.ProjectDiscovery),
	}
}

func targetDetails(options HealthCheckOptions) TargetDetails {
	if options.Target == nil {
		return TargetDetails{}
	}
	return TargetDetails{
		Destination: options.Target.String(),
		IsLocalhost: options.Target.IsPlainLocalhost(),
	}
}

func toReadinessReport(evaluatedHealthCheck EvaluatedReadinessCheck) ReadinessReport {
	return ReadinessReport{
		Host:   toDependencyReports(evaluatedHealthCheck.Host),
		Target: toDependencyReports(evaluatedHealthCheck.Target),
	}
}

func ToDependencyReport(status EvaluatedDependency) DependencyReport {
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

func toDependencyReports(statuses []EvaluatedDependency) []DependencyReport {
	reports := make([]DependencyReport, len(statuses))
	for i, status := range statuses {
		reports[i] = ToDependencyReport(status)
	}
	return reports
}
