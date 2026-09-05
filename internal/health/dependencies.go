package health

import (
	"context"
	"fmt"

	"github.com/arm/topo/internal/runner"
	"github.com/arm/topo/internal/ssh"
	"github.com/arm/topo/internal/upgrade"
	"github.com/arm/topo/internal/version"
)

type WarningError struct{ Err error }

func (w WarningError) Error() string { return w.Err.Error() }

type InfoError struct{ Err error }

func (i InfoError) Error() string { return i.Err.Error() }

type HardwareCapability int

const (
	Remoteproc HardwareCapability = iota
)

const containerEngineInstallURL = "https://github.com/arm/topo#install-a-container-engine"

type DependencyID string

type Dependency struct {
	ID                    DependencyID
	Binary                string
	Label                 string
	Checks                []Check
	SoftwarePrerequisites []DependencyID
	HardwarePrerequisites []HardwareCapability
}

func HostRequiredDependencies() []Dependency {
	topo := Dependency{
		ID:     DependencyID("topo"),
		Binary: "topo",
		Label:  "Topo",
		Checks: []Check{VersionMatches{
			FetchLatest: func(ctx context.Context) (string, error) {
				if version.Version == version.Dev {
					return version.Version, nil
				}

				binPath, err := upgrade.CurrentBinaryPath()
				if err == nil && upgrade.IsBinaryManagedByHomebrew(binPath) {
					return version.FetchLatestHomebrew(ctx, version.HomebrewFormulaURL)
				}

				return version.FetchLatestArtifactory(ctx, version.ArtifactoryBaseURL)
			},
			CurrentVersion: version.Version,
			BuildFix: func() Fix {
				fix := Fix{
					Description: "Upgrade Topo",
				}

				binPath, err := upgrade.CurrentBinaryPath()
				if err != nil {
					return fix
				}

				_, fix.Command = upgrade.GetUpgradeCommand(binPath)
				return fix
			},
		}},
	}

	ssh := Dependency{
		ID:     DependencyID("ssh"),
		Binary: "ssh",
		Label:  "OpenSSH",
		Checks: []Check{BinaryExists{}, OpenSSHAvailable{}},
	}

	docker := Dependency{
		ID:     DependencyID("host-docker"),
		Binary: "docker",
		Label:  "Container Engine",
		Checks: []Check{
			BinaryExists{
				Fix: &Fix{
					Description: "Install a supported container engine. See " + containerEngineInstallURL,
				},
			},
			CommandSuccessful{
				Cmd: "docker info",
				Fix: &Fix{
					Description: "Ensure current user can run docker commands. See " + containerEngineInstallURL,
				},
			},
		},
	}

	dockerCompose := Dependency{
		ID:     DependencyID("docker-compose"),
		Binary: "docker-compose",
		Label:  "Docker Compose",
		Checks: []Check{
			CommandSuccessful{
				Cmd: "docker compose",
				Fix: &Fix{
					Description: "Ensure Docker Compose is installed as a plugin for Docker. See " + containerEngineInstallURL,
				},
			},
			DockerComposeMinVersion{
				MinVersion: "2.21.0",
			},
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
		Checks: []Check{
			BinaryExists{
				Fix: &Fix{
					Description: "Install a supported container engine. See " + containerEngineInstallURL,
				},
			},
			CommandSuccessful{
				Cmd: "docker info",
				Fix: &Fix{
					Description: "Ensure current user can run docker commands. See " + containerEngineInstallURL,
				},
			},
		},
	}

	remoteprocRuntime := Dependency{
		ID:                    DependencyID("remoteproc-runtime"),
		Binary:                "remoteproc-runtime",
		Label:                 "Remoteproc Runtime",
		SoftwarePrerequisites: []DependencyID{docker.ID},
		HardwarePrerequisites: []HardwareCapability{Remoteproc},
		Checks: []Check{
			BinaryExists{
				Severity: SeverityWarning,
				Fix: &Fix{
					Description: "Install the Remoteproc Runtime",
					Command:     fmt.Sprintf("topo install remoteproc-runtime --target %s", target),
				},
			},
		},
	}
	remoteprocRuntimeShim := Dependency{
		ID:                    DependencyID("containerd-shim-remoteproc-v1"),
		Binary:                "containerd-shim-remoteproc-v1",
		Label:                 "Remoteproc Shim",
		SoftwarePrerequisites: []DependencyID{docker.ID},
		HardwarePrerequisites: []HardwareCapability{Remoteproc},
		Checks: []Check{
			BinaryExists{
				Severity: SeverityWarning,
				Fix: &Fix{
					Description: "Install the Remoteproc Runtime",
					Command:     fmt.Sprintf("topo install remoteproc-runtime --target %s", target),
				},
			},
		},
	}

	lscpu := Dependency{
		ID:     DependencyID("lscpu"),
		Binary: "lscpu",
		Label:  "Hardware Info",
		Checks: []Check{BinaryExists{}},
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
	Error      error
	Fix        *Fix
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

		var fix *Fix
		var err error
		for _, check := range dep.Checks {
			fix, err = check.Run(ctx, runner, dep)
			if err != nil {
				break
			}
		}

		if err == nil {
			healthy[dep.ID] = struct{}{}
		}

		result = append(result, DependencyStatus{
			Dependency: dep,
			Error:      err,
			Fix:        fix,
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
