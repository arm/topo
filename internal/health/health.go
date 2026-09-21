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
	TargetDetails    *TargetDetails
	Deployment       ReadinessReport
	ProjectDiscovery ReadinessReport
}

func Check(ctx context.Context, options HealthCheckOptions) HealthReport {
	healthCheck := NewHealthCheck(options)
	evaluatedHealthCheck := healthCheck.Evaluate(ctx)
	return evaluatedHealthCheck.Report(targetDetails(options), options.MissingTargetFixMessage)
}

func (h EvaluatedHealthCheck) Report(target *TargetDetails, missingTargetFixMessage string) HealthReport {
	deployment := toReadinessReport(h.Deployment)
	discovery := toReadinessReport(h.ProjectDiscovery)
	if target == nil {
		deployment.Target = append(deployment.Target, missingTargetReport(SeverityError, "target not specified", missingTargetFixMessage))
		discovery.Target = append(discovery.Target, missingTargetReport(SeverityWarning, "target not specified; cannot calculate project compatibility", missingTargetFixMessage))
	} else {
		deployment.Target = removeSuccessfulLocalhostConnectivityReports(deployment.Target, target)
		discovery.Target = removeSuccessfulLocalhostConnectivityReports(discovery.Target, target)
	}
	return HealthReport{
		TargetDetails:    target,
		Deployment:       deployment,
		ProjectDiscovery: discovery,
	}
}

func missingTargetReport(severity CheckSeverity, message, fixMessage string) DependencyReport {
	report := DependencyReport{Name: "Target", Status: checkStatusFromSeverity(severity), Value: message}
	if fixMessage != "" {
		report.Fix = &Fix{Description: fixMessage}
	}
	return report
}

func removeSuccessfulLocalhostConnectivityReports(reports []DependencyReport, target *TargetDetails) []DependencyReport {
	return slices.DeleteFunc(reports, func(report DependencyReport) bool {
		return report.Status == CheckStatusOK && report.ID == DependencyIDConnectivity && target.IsLocalhost
	})
}

func targetDetails(options HealthCheckOptions) *TargetDetails {
	if options.Target == nil {
		return nil
	}
	return &TargetDetails{
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
