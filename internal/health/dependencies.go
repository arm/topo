package health

import (
	"context"
	"fmt"
	"strings"

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

type SoftwareDependency int

const (
	UnsetSoftwareDependency SoftwareDependency = iota
	Docker
	Lscpu
)

type Dependency struct {
	Binary                string
	Label                 string
	Checks                []Check
	SoftwareEnumID        SoftwareDependency
	SoftwarePrerequisites []SoftwareDependency
	HardwarePrerequisite  []HardwareCapability
}

type OpenSSHAvailable struct{}

func (o OpenSSHAvailable) Run(ctx context.Context, r runner.Runner, dep Dependency) (*Fix, error) {
	_, stderr, err := r.Run(ctx, "ssh -V")
	if err != nil {
		return nil, err
	}
	if !strings.Contains(stderr, "OpenSSH_") {
		return &Fix{
			Description: "Install OpenSSH and ensure its ssh executable is first on PATH",
		}, fmt.Errorf("%q does not resolve to OpenSSH: %s", dep.Binary, stderr)
	}
	return nil, nil
}

var HostRequiredDependencies = []Dependency{
	{
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
	},
	{
		Binary: "ssh",
		Label:  "OpenSSH",
		Checks: []Check{BinaryExists{}, OpenSSHAvailable{}},
	},
	{
		Binary: "curl",
		Label:  "Curl",
		Checks: []Check{
			BinaryExists{},
		},
	},
	{
		Binary:         "docker",
		Label:          "Container Engine",
		SoftwareEnumID: Docker,
		Checks: []Check{
			BinaryExists{},
			CommandSuccessful{
				Cmd: "docker info",
				Fix: &Fix{Description: "Ensure current user can run docker commands"},
			},
		},
	},
}

func TargetRequiredDependencies(target ssh.Destination) []Dependency {
	return []Dependency{
		{
			Binary:         "docker",
			Label:          "Container Engine",
			SoftwareEnumID: Docker,
			Checks: []Check{
				BinaryExists{},
				CommandSuccessful{
					Cmd: "docker info",
					Fix: &Fix{Description: "Ensure current user can run docker commands"},
				},
			},
		},
		{
			Binary:                "remoteproc-runtime",
			Label:                 "Remoteproc Runtime",
			SoftwarePrerequisites: []SoftwareDependency{Docker},
			HardwarePrerequisite:  []HardwareCapability{Remoteproc},
			Checks: []Check{
				BinaryExists{
					Severity: SeverityWarning,
					Fix: &Fix{
						Description: "Install the Remoteproc Runtime",
						Command:     fmt.Sprintf("topo install remoteproc-runtime --target %s", target),
					},
				},
			},
		},
		{
			Binary:                "containerd-shim-remoteproc-v1",
			Label:                 "Remoteproc Shim",
			SoftwarePrerequisites: []SoftwareDependency{Docker},
			HardwarePrerequisite:  []HardwareCapability{Remoteproc},
			Checks: []Check{
				BinaryExists{
					Severity: SeverityWarning,
					Fix: &Fix{
						Description: "Install the Remoteproc Runtime",
						Command:     fmt.Sprintf("topo install remoteproc-runtime --target %s", target),
					},
				},
			},
		},
		{
			Binary:         "lscpu",
			Label:          "Hardware Info",
			SoftwareEnumID: Lscpu,
			Checks:         []Check{BinaryExists{}},
		},
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
		if len(dep.HardwarePrerequisite) == 0 || hardwareCapabilityMatches(dep.HardwarePrerequisite, hardware) {
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
	installed := make(map[SoftwareDependency]struct{})
	result := make([]DependencyStatus, 0, len(dependencies))

	for _, dep := range dependencies {
		if len(dep.SoftwarePrerequisites) > 0 && !hasAnyInstalledPrerequisite(dep.SoftwarePrerequisites, installed) {
			continue
		}

		var fix *Fix
		var err error
		for _, check := range dep.Checks {
			fix, err = check.Run(ctx, runner, dep)
			if _, ok := check.(BinaryExists); ok && err == nil {
				installed[dep.SoftwareEnumID] = struct{}{}
			}

			if err != nil {
				break
			}
		}

		result = append(result, DependencyStatus{
			Dependency: dep,
			Error:      err,
			Fix:        fix,
		})
	}
	return result
}

func hasAnyInstalledPrerequisite(required []SoftwareDependency, installed map[SoftwareDependency]struct{}) bool {
	for _, softwareDep := range required {
		if _, exists := installed[softwareDep]; exists {
			return true
		}
	}
	return false
}
