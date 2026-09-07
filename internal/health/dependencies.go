package health

import (
	"context"
	"fmt"

	"github.com/arm/topo/internal/runner"
	"github.com/arm/topo/internal/ssh"
)

type HardwareCapability int

const (
	Remoteproc HardwareCapability = iota
)

const containerEngineInstallURL = "https://github.com/arm/topo#install-a-container-engine"

type DependencyID string

type CheckResult struct {
	SuccessValue string
	Failure      *CheckFailure
}

type DependencyCheckFn func(ctx context.Context, r runner.Runner) CheckResult

type Dependency struct {
	ID                    DependencyID
	Label                 string
	Check                 DependencyCheckFn
	SoftwarePrerequisites []DependencyID
	HardwarePrerequisites []HardwareCapability
}

func HostRequiredDependencies(skipVersionChecks bool) []Dependency {
	topo := Dependency{
		ID:    DependencyID("topo"),
		Label: "Topo",
		Check: func(ctx context.Context, _ runner.Runner) CheckResult {
			if skipVersionChecks {
				return CheckResult{SuccessValue: "topo"}
			}
			if failure := CheckTopoIsUpToDate(ctx); failure != nil {
				return CheckResult{Failure: failure}
			}
			return CheckResult{SuccessValue: "topo"}
		},
	}

	ssh := Dependency{
		ID:    DependencyID("ssh"),
		Label: "OpenSSH",
		Check: func(ctx context.Context, r runner.Runner) CheckResult {
			if err := r.BinaryExists(ctx, "ssh"); err != nil {
				return CheckResult{Failure: &CheckFailure{Severity: SeverityError, Message: err.Error()}}
			}
			if failure := CheckOpenSSHAvailable(ctx, r, "ssh"); failure != nil {
				return CheckResult{Failure: failure}
			}
			return CheckResult{SuccessValue: "ssh"}
		},
	}

	docker := Dependency{
		ID:    DependencyID("host-docker"),
		Label: "Container Engine",
		Check: func(ctx context.Context, r runner.Runner) CheckResult {
			if err := r.BinaryExists(ctx, "docker"); err != nil {
				return CheckResult{Failure: &CheckFailure{
					Severity: SeverityError,
					Message:  err.Error(),
					Fix:      &Fix{Description: "Install a supported container engine. See " + containerEngineInstallURL},
				}}
			}
			if _, _, err := r.Run(ctx, "docker info"); err != nil {
				return CheckResult{Failure: &CheckFailure{
					Severity: SeverityError,
					Message:  err.Error(),
					Fix:      &Fix{Description: "Ensure current user can run docker commands. See " + containerEngineInstallURL},
				}}
			}
			return CheckResult{SuccessValue: "docker"}
		},
	}

	dockerCompose := Dependency{
		ID:    DependencyID("docker-compose"),
		Label: "Docker Compose",
		Check: func(ctx context.Context, r runner.Runner) CheckResult {
			if _, _, err := r.Run(ctx, "docker compose"); err != nil {
				return CheckResult{Failure: &CheckFailure{
					Severity: SeverityError,
					Message:  err.Error(),
					Fix:      &Fix{Description: "Ensure Docker Compose is installed as a plugin for Docker. See " + containerEngineInstallURL},
				}}
			}
			if failure := CheckDockerComposeMinVersion(ctx, r, "2.21.0"); failure != nil {
				return CheckResult{Failure: failure}
			}
			return CheckResult{SuccessValue: "docker compose"}
		},
		SoftwarePrerequisites: []DependencyID{docker.ID},
	}

	return []Dependency{topo, ssh, docker, dockerCompose}
}

func TargetRequiredDependencies(target ssh.Destination) []Dependency {
	docker := Dependency{
		ID:    DependencyID("target-docker"),
		Label: "Container Engine",
		Check: func(ctx context.Context, r runner.Runner) CheckResult {
			if err := r.BinaryExists(ctx, "docker"); err != nil {
				return CheckResult{Failure: &CheckFailure{
					Severity: SeverityError,
					Message:  err.Error(),
					Fix:      &Fix{Description: "Install a supported container engine. See " + containerEngineInstallURL},
				}}
			}
			if _, _, err := r.Run(ctx, "docker info"); err != nil {
				return CheckResult{Failure: &CheckFailure{
					Severity: SeverityError,
					Message:  err.Error(),
					Fix:      &Fix{Description: "Ensure current user can run docker commands. See " + containerEngineInstallURL},
				}}
			}
			return CheckResult{SuccessValue: "docker"}
		},
	}

	remoteprocRuntime := Dependency{
		ID:                    DependencyID("remoteproc-runtime"),
		Label:                 "Remoteproc Runtime",
		SoftwarePrerequisites: []DependencyID{docker.ID},
		HardwarePrerequisites: []HardwareCapability{Remoteproc},
		Check: func(ctx context.Context, r runner.Runner) CheckResult {
			if err := r.BinaryExists(ctx, "remoteproc-runtime"); err != nil {
				return CheckResult{Failure: &CheckFailure{
					Severity: SeverityWarning,
					Message:  err.Error(),
					Fix: &Fix{
						Description: "Install the Remoteproc Runtime",
						Command:     fmt.Sprintf("topo install remoteproc-runtime --target %s", target),
					},
				}}
			}
			return CheckResult{SuccessValue: "remoteproc-runtime"}
		},
	}

	remoteprocRuntimeShim := Dependency{
		ID:                    DependencyID("containerd-shim-remoteproc-v1"),
		Label:                 "Remoteproc Shim",
		SoftwarePrerequisites: []DependencyID{docker.ID},
		HardwarePrerequisites: []HardwareCapability{Remoteproc},
		Check: func(ctx context.Context, r runner.Runner) CheckResult {
			if err := r.BinaryExists(ctx, "containerd-shim-remoteproc-v1"); err != nil {
				return CheckResult{Failure: &CheckFailure{
					Severity: SeverityWarning,
					Message:  err.Error(),
					Fix: &Fix{
						Description: "Install the Remoteproc Runtime",
						Command:     fmt.Sprintf("topo install remoteproc-runtime --target %s", target),
					},
				}}
			}
			return CheckResult{SuccessValue: "containerd-shim-remoteproc-v1"}
		},
	}

	lscpu := Dependency{
		ID:    DependencyID("lscpu"),
		Label: "Hardware Info",
		Check: func(ctx context.Context, r runner.Runner) CheckResult {
			if err := r.BinaryExists(ctx, "lscpu"); err != nil {
				return CheckResult{Failure: &CheckFailure{Severity: SeverityError, Message: err.Error()}}
			}
			return CheckResult{SuccessValue: "lscpu"}
		},
	}

	return []Dependency{docker, remoteprocRuntime, remoteprocRuntimeShim, lscpu}
}

type DependencyStatus struct {
	Dependency Dependency
	Result     CheckResult
}

func FilterByHardware(deps []Dependency, hardware map[HardwareCapability]struct{}) []Dependency {
	result := make([]Dependency, 0, len(deps))
	for _, dep := range deps {
		if len(dep.HardwarePrerequisites) == 0 || hardwareCapabilityMatches(dep.HardwarePrerequisites, hardware) {
			result = append(result, dep)
		}
	}
	return result
}

func hardwareCapabilityMatches(required []HardwareCapability, available map[HardwareCapability]struct{}) bool {
	for _, capability := range required {
		if _, exists := available[capability]; exists {
			return true
		}
	}
	return false
}

func PerformChecks(ctx context.Context, dependencies []Dependency, runner runner.Runner) []DependencyStatus {
	healthy := make(map[DependencyID]struct{})
	result := make([]DependencyStatus, 0, len(dependencies))

	for _, dep := range dependencies {
		if !allPrerequisitesFulfilled(dep.SoftwarePrerequisites, healthy) {
			continue
		}

		checkResult := CheckResult{}
		if dep.Check != nil {
			checkResult = dep.Check(ctx, runner)
		}
		if checkResult.Failure == nil {
			healthy[dep.ID] = struct{}{}
		}

		result = append(result, DependencyStatus{Dependency: dep, Result: checkResult})
	}
	return result
}

func allPrerequisitesFulfilled(required []DependencyID, healthy map[DependencyID]struct{}) bool {
	for _, dep := range required {
		if _, ok := healthy[dep]; !ok {
			return false
		}
	}
	return true
}
