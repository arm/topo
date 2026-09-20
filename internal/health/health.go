package health

import (
	"context"
	"slices"
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
	deployment := toReadinessReport(h.Deployment)
	discovery := toReadinessReport(h.ProjectDiscovery)
	downgradeMissingTargetForProjectDiscovery(discovery.Target)
	deployment.Target = removeSuccessfulTargetPrerequisiteReports(deployment.Target, target)
	discovery.Target = removeSuccessfulTargetPrerequisiteReports(discovery.Target, target)
	return HealthReport{
		TargetDetails:    target,
		Deployment:       deployment,
		ProjectDiscovery: discovery,
	}
}

func downgradeMissingTargetForProjectDiscovery(reports []DependencyReport) {
	for i := range reports {
		report := &reports[i]
		if report.ID == DependencyIDTargetSpecified && report.Status == CheckStatusError {
			report.Status = CheckStatusWarning
			report.Value = "target not specified; cannot calculate project compatibility"
			return
		}
	}
}

func removeSuccessfulTargetPrerequisiteReports(reports []DependencyReport, target TargetDetails) []DependencyReport {
	return slices.DeleteFunc(reports, func(report DependencyReport) bool {
		isOK := report.Status == CheckStatusOK
		isTargetSpecifiedCheck := report.ID == DependencyIDTargetSpecified
		isLocalhostConnectivityCheck := report.ID == DependencyIDConnectivity && target.IsLocalhost
		return isOK && (isTargetSpecifiedCheck || isLocalhostConnectivityCheck)
	})
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
