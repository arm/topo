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
	return evaluatedHealthCheck.Report(targetDetails(options))
}

func (h EvaluatedHealthCheck) Report(target TargetDetails) HealthReport {
	deployment := toReadinessReport(h.Deployment, target)
	discovery := toReadinessReport(h.ProjectDiscovery, target)
	for i := range discovery.Target {
		report := &discovery.Target[i]
		if report.ID == DependencyIDTargetSpecified && report.Status == CheckStatusError {
			report.Status = CheckStatusWarning
			report.Value = "target not specified; cannot calculate project compatibility"
		}
	}
	return HealthReport{
		TargetDetails:    target,
		Deployment:       deployment,
		ProjectDiscovery: discovery,
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

func toReadinessReport(evaluatedHealthCheck EvaluatedReadinessCheck, target TargetDetails) ReadinessReport {
	targetReports := make([]DependencyReport, 0, len(evaluatedHealthCheck.Target))
	for _, dependency := range evaluatedHealthCheck.Target {
		// Keep prerequisite successes in the evaluation without adding report noise.
		if dependency.Result.Failure == nil && (dependency.ID == DependencyIDTargetSpecified ||
			(dependency.ID == DependencyIDConnectivity && target.IsLocalhost)) {
			continue
		}
		targetReports = append(targetReports, ToDependencyReport(dependency))
	}
	return ReadinessReport{
		Host:   toDependencyReports(evaluatedHealthCheck.Host),
		Target: targetReports,
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
