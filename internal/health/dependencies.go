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

type DependencyCheck func(ctx context.Context, r runner.Runner) *CheckFailure

type Dependency struct {
	ID                    DependencyID
	Binary                string
	Label                 string
	Check                 DependencyCheck
	SoftwarePrerequisites []DependencyID
	HardwarePrerequisites []HardwareCapability
}

func HostRequiredDependencies(skipVersionChecks bool) []Dependency {
	topo := Dependency{
		ID:     DependencyID("topo"),
		Binary: "topo",
		Label:  "Topo",
		Check: func(ctx context.Context, _ runner.Runner) *CheckFailure {
			if skipVersionChecks {
				return nil
			}
			return CheckTopoIsUpToDate(ctx)
		},
	}

	ssh := Dependency{
		ID:     DependencyID("ssh"),
		Binary: "ssh",
		Label:  "OpenSSH",
		Check: func(ctx context.Context, r runner.Runner) *CheckFailure {
			if err := r.BinaryExists(ctx, "ssh"); err != nil {
				return &CheckFailure{Severity: SeverityError, Message: err.Error()}
			}
			return CheckOpenSSHAvailable(ctx, r, "ssh")
		},
	}

	docker := Dependency{
		ID:     DependencyID("host-docker"),
		Binary: "docker",
		Label:  "Container Engine",
		Check: func(ctx context.Context, r runner.Runner) *CheckFailure {
			if err := r.BinaryExists(ctx, "docker"); err != nil {
				return &CheckFailure{
					Severity: SeverityError,
					Message:  err.Error(),
					Fix: &Fix{
						Description: "Install a supported container engine. See " + containerEngineInstallURL,
					},
				}
			}
			if err := CheckCommandSuccessful(ctx, r, "docker info"); err != nil {
				return &CheckFailure{
					Severity: SeverityError,
					Message:  err.Error(),
					Fix: &Fix{
						Description: "Ensure current user can run docker commands. See " + containerEngineInstallURL,
					},
				}
			}
			return nil
		},
	}

	dockerCompose := Dependency{
		ID:     DependencyID("docker-compose"),
		Binary: "docker-compose",
		Label:  "Docker Compose",
		Check: func(ctx context.Context, r runner.Runner) *CheckFailure {
			if err := CheckCommandSuccessful(ctx, r, "docker compose"); err != nil {
				return &CheckFailure{
					Severity: SeverityError,
					Message:  err.Error(),
					Fix: &Fix{
						Description: "Ensure Docker Compose is installed as a plugin for Docker. See " + containerEngineInstallURL,
					},
				}
			}
			return CheckDockerComposeMinVersion(ctx, r, "2.21.0")
		},
		SoftwarePrerequisites: []DependencyID{docker.ID},
	}

	return []Dependency{
		topo,
		ssh,
		docker,
		dockerCompose,
	}
}

func TargetRequiredDependencies(target ssh.Destination) []Dependency {
	docker := Dependency{
		ID:     DependencyID("target-docker"),
		Binary: "docker",
		Label:  "Container Engine",
		Check: func(ctx context.Context, r runner.Runner) *CheckFailure {
			if err := r.BinaryExists(ctx, "docker"); err != nil {
				return &CheckFailure{
					Severity: SeverityError,
					Message:  err.Error(),
					Fix: &Fix{
						Description: "Install a supported container engine. See " + containerEngineInstallURL,
					},
				}
			}
			if err := CheckCommandSuccessful(ctx, r, "docker info"); err != nil {
				return &CheckFailure{
					Severity: SeverityError,
					Message:  err.Error(),
					Fix: &Fix{
						Description: "Ensure current user can run docker commands. See " + containerEngineInstallURL,
					},
				}
			}
			return nil
		},
	}

	remoteprocRuntime := Dependency{
		ID:                    DependencyID("remoteproc-runtime"),
		Binary:                "remoteproc-runtime",
		Label:                 "Remoteproc Runtime",
		SoftwarePrerequisites: []DependencyID{docker.ID},
		HardwarePrerequisites: []HardwareCapability{Remoteproc},
		Check: func(ctx context.Context, r runner.Runner) *CheckFailure {
			if err := r.BinaryExists(ctx, "remoteproc-runtime"); err != nil {
				return &CheckFailure{
					Severity: SeverityWarning,
					Message:  err.Error(),
					Fix: &Fix{
						Description: "Install the Remoteproc Runtime",
						Command:     fmt.Sprintf("topo install remoteproc-runtime --target %s", target),
					},
				}
			}
			return nil
		},
	}
	remoteprocRuntimeShim := Dependency{
		ID:                    DependencyID("containerd-shim-remoteproc-v1"),
		Binary:                "containerd-shim-remoteproc-v1",
		Label:                 "Remoteproc Shim",
		SoftwarePrerequisites: []DependencyID{docker.ID},
		HardwarePrerequisites: []HardwareCapability{Remoteproc},
		Check: func(ctx context.Context, r runner.Runner) *CheckFailure {
			if err := r.BinaryExists(ctx, "containerd-shim-remoteproc-v1"); err != nil {
				return &CheckFailure{
					Severity: SeverityWarning,
					Message:  err.Error(),
					Fix: &Fix{
						Description: "Install the Remoteproc Runtime",
						Command:     fmt.Sprintf("topo install remoteproc-runtime --target %s", target),
					},
				}
			}
			return nil
		},
	}

	lscpu := Dependency{
		ID:     DependencyID("lscpu"),
		Binary: "lscpu",
		Label:  "Hardware Info",
		Check: func(ctx context.Context, r runner.Runner) *CheckFailure {
			if err := r.BinaryExists(ctx, "lscpu"); err != nil {
				return &CheckFailure{Severity: SeverityError, Message: err.Error()}
			}
			return nil
		},
	}

	return []Dependency{
		docker,
		remoteprocRuntime,
		remoteprocRuntimeShim,
		lscpu,
	}
}

type DependencyStatus struct {
	Dependency Dependency
	Failure    *CheckFailure
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

		var failure *CheckFailure
		if dep.Check != nil {
			failure = dep.Check(ctx, runner)
		}

		if failure == nil {
			healthy[dep.ID] = struct{}{}
		}

		result = append(result, DependencyStatus{
			Dependency: dep,
			Failure:    failure,
		})
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
