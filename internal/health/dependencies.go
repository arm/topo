package health

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/arm/topo/internal/command"
	"github.com/arm/topo/internal/output/logger"
	"github.com/arm/topo/internal/probe"
	"github.com/arm/topo/internal/runner"
	"github.com/arm/topo/internal/ssh"
	"github.com/arm/topo/internal/upgrade"
	"github.com/arm/topo/internal/version"
)

const containerEngineInstallURL = "https://github.com/arm/topo#install-a-container-engine"

type DependencyID string

const (
	DependencyIDConnectivity DependencyID = "target-connectivity"
	DependencyIDRemoteproc   DependencyID = "remoteproc"
)

type Dependency struct {
	Label string
	Check DependencyCheckFn
	// Used to maintain legacy JSON output
	ID DependencyID
}

type DependencyCheckFn func(ctx context.Context) DependencyCheckResult

type DependencyCheckResult struct {
	SuccessValue string
	Failure      *DependencyCheckFailure
}

type DependencyCheckFailure struct {
	Severity CheckSeverity
	Message  string
	Fix      *Fix
}

type CheckSeverity int

const (
	SeverityError CheckSeverity = iota
	SeverityWarning
	SeverityInfo
)

type Fix struct {
	Description string
	Command     string
}

func NewDependencyOnSSH(r runner.Runner) Dependency {
	return Dependency{
		Label: "OpenSSH",
		Check: func(ctx context.Context) DependencyCheckResult {
			if err := r.BinaryExists(ctx, "ssh"); err != nil {
				return DependencyCheckResult{Failure: &DependencyCheckFailure{Severity: SeverityError, Message: err.Error()}}
			}
			_, stderr, err := r.Run(ctx, "ssh -V")
			if err != nil {
				return DependencyCheckResult{Failure: &DependencyCheckFailure{Message: err.Error()}}
			}
			if !strings.Contains(stderr, "OpenSSH_") {
				return DependencyCheckResult{Failure: &DependencyCheckFailure{
					Message: fmt.Sprintf("%q does not resolve to OpenSSH: %s", "ssh", stderr),
					Fix: &Fix{
						Description: "Install OpenSSH and ensure its ssh executable is first on PATH",
					},
				}}
			}
			return DependencyCheckResult{SuccessValue: "ssh"}
		},
	}
}

func NewDependencyOnTopo(skipVersionChecks bool) Dependency {
	return Dependency{
		Label: "Topo",
		Check: func(ctx context.Context) DependencyCheckResult {
			if skipVersionChecks || version.Version == version.Dev {
				return DependencyCheckResult{SuccessValue: "topo"}
			}
			binPath, binPathErr := upgrade.CurrentBinaryPath()

			var latest string
			var err error
			if binPathErr == nil && upgrade.IsBinaryManagedByHomebrew(binPath) {
				latest, err = version.FetchLatestHomebrew(ctx, version.HomebrewFormulaURL)
			} else {
				latest, err = version.FetchLatestArtifactory(ctx, version.ArtifactoryBaseURL)
			}
			if err != nil {
				logger.Warn(fmt.Sprintf("failed to fetch latest version: %v", err))
				return DependencyCheckResult{SuccessValue: "topo"}
			}
			if latest == version.Version {
				return DependencyCheckResult{SuccessValue: "topo"}
			}

			fix := Fix{Description: "Upgrade Topo"}
			if binPathErr == nil {
				_, fix.Command = upgrade.GetUpgradeCommand(binPath)
			}
			return DependencyCheckResult{Failure: &DependencyCheckFailure{
				Severity: SeverityInfo,
				Message:  fmt.Sprintf("out of date - current: %s, latest version: %s", version.Version, latest),
				Fix:      &fix,
			}}
		},
	}
}

func NewDependencyOnDocker(r runner.Runner) Dependency {
	return Dependency{
		Label: "Container Engine",
		Check: func(ctx context.Context) DependencyCheckResult {
			if err := r.BinaryExists(ctx, "docker"); err != nil {
				return DependencyCheckResult{Failure: &DependencyCheckFailure{
					Severity: SeverityError,
					Message:  err.Error(),
					Fix:      &Fix{Description: "Install a supported container engine. See " + containerEngineInstallURL},
				}}
			}
			if _, _, err := r.Run(ctx, "docker info"); err != nil {
				return DependencyCheckResult{Failure: &DependencyCheckFailure{
					Severity: SeverityError,
					Message:  err.Error(),
					Fix:      &Fix{Description: "Ensure current user can run docker commands. See " + containerEngineInstallURL},
				}}
			}
			return DependencyCheckResult{SuccessValue: "docker"}
		},
	}
}

func NewDependencyOnRemoteDocker(hostRunner runner.Runner, probeInfo func(context.Context) error) Dependency {
	return Dependency{
		Label: "Container Engine",
		Check: func(ctx context.Context) DependencyCheckResult {
			if err := hostRunner.BinaryExists(ctx, "docker"); err != nil {
				return DependencyCheckResult{Failure: &DependencyCheckFailure{
					Severity: SeverityError,
					Message:  fmt.Sprintf("cannot probe from host: %v", err),
					Fix:      &Fix{Description: "Install a supported container engine on the host. See " + containerEngineInstallURL},
				}}
			}
			if err := probeInfo(ctx); err != nil {
				return DependencyCheckResult{Failure: &DependencyCheckFailure{
					Severity: SeverityError,
					Message:  err.Error(),
					Fix:      &Fix{Description: "Ensure docker is installed and running on the target. See " + containerEngineInstallURL},
				}}
			}
			return DependencyCheckResult{SuccessValue: "docker"}
		},
	}
}

func NewDependencyOnDockerCompose(r runner.Runner) Dependency {
	return Dependency{
		Label: "Docker Compose",
		Check: func(ctx context.Context) DependencyCheckResult {
			stdout, _, err := r.Run(ctx, "docker compose version --format json")
			if err != nil {
				return DependencyCheckResult{Failure: &DependencyCheckFailure{
					Severity: SeverityError,
					Message:  err.Error(),
					Fix:      &Fix{Description: "Ensure Docker Compose is installed as a plugin for Docker. See " + containerEngineInstallURL},
				}}
			}

			var output struct {
				Version string `json:"version"`
			}
			if err := json.Unmarshal([]byte(stdout), &output); err != nil {
				return DependencyCheckResult{Failure: &DependencyCheckFailure{Message: err.Error()}}
			}
			if !version.IsAtLeastVersion(output.Version, "2.21.0") {
				return DependencyCheckResult{Failure: &DependencyCheckFailure{
					Message: fmt.Sprintf("installed docker compose version %s is older than required version %s", output.Version, "2.21.0"),
					Fix: &Fix{
						Description: fmt.Sprintf("Upgrade Docker Compose to version %s or later. See %s", "2.21.0", containerEngineInstallURL),
					},
				}}
			}

			return DependencyCheckResult{SuccessValue: "docker compose"}
		},
	}
}

func NewMissingTargetDependency(message string, severity CheckSeverity, fixMessage string) Dependency {
	return Dependency{
		ID:    DependencyIDConnectivity,
		Label: "Connectivity",
		Check: func(context.Context) DependencyCheckResult {
			failure := &DependencyCheckFailure{Severity: severity, Message: message}
			if fixMessage != "" {
				failure.Fix = &Fix{Description: fixMessage}
			}
			return DependencyCheckResult{Failure: failure}
		},
	}
}

type ConnectivityOperations struct {
	Authenticate    func(context.Context) error
	KnownHostsEntry func() (string, error)
}

func NewConnectivityDependency(target ssh.Destination, operations ConnectivityOperations) Dependency {
	return Dependency{
		ID:    DependencyIDConnectivity,
		Label: "Connectivity",
		Check: func(ctx context.Context) DependencyCheckResult {
			err := operations.Authenticate(ctx)
			if err == nil {
				return DependencyCheckResult{SuccessValue: target.String()}
			}

			failure := DependencyCheckFailure{Severity: SeverityError, Message: err.Error()}
			switch {
			case errors.Is(err, ssh.ErrAuthFailed), errors.Is(err, ssh.ErrTooManyAuthFails):
				failure.Fix = &Fix{
					Description: "Configure SSH keys on remote target",
					Command:     fmt.Sprintf("topo setup-keys --target %s", target),
				}
			case errors.Is(err, ssh.ErrHostKeyUnknown):
				failure.Fix = &Fix{
					Description: "Trust the target's SSH host key",
					Command:     fmt.Sprintf("topo health --target %s --accept-new-host-keys", target),
				}
			case errors.Is(err, ssh.ErrHostKeyChanged):
				knownHostsEntry, configErr := operations.KnownHostsEntry()
				fixCommand := ""
				if configErr == nil {
					fixCommand = fmt.Sprintf("ssh-keygen -R %s", command.QuoteArg(knownHostsEntry))
				}
				failure.Fix = &Fix{
					Description: "Remove the old SSH host key from known_hosts, then retry",
					Command:     fixCommand,
				}
			}
			return DependencyCheckResult{Failure: &failure}
		},
	}
}

func NewDependencyOnRemoteproc(r runner.Runner) Dependency {
	return Dependency{
		ID:    DependencyIDRemoteproc,
		Label: "Processing Domain Driver (remoteproc)",
		Check: func(ctx context.Context) DependencyCheckResult {
			remoteProcessors, err := probe.Remoteproc(ctx, r)
			if err != nil {
				return DependencyCheckResult{
					Failure: &DependencyCheckFailure{
						Severity: SeverityError,
						Message:  err.Error(),
					},
				}
			}
			if len(remoteProcessors) > 0 {
				names := make([]string, len(remoteProcessors))
				for i, remoteProc := range remoteProcessors {
					names[i] = remoteProc.Name
				}
				return DependencyCheckResult{
					SuccessValue: strings.Join(names, ", "),
				}
			}
			return DependencyCheckResult{
				Failure: &DependencyCheckFailure{
					Severity: SeverityInfo,
					Message:  "no remoteproc devices found",
				},
			}
		},
	}
}

func NewDependencyOnRemoteprocRuntime(target ssh.Destination, r runner.Runner) Dependency {
	return Dependency{
		Label: "Remoteproc Runtime",
		Check: func(ctx context.Context) DependencyCheckResult {
			if err := r.BinaryExists(ctx, "remoteproc-runtime"); err != nil {
				return DependencyCheckResult{Failure: &DependencyCheckFailure{
					Severity: SeverityWarning,
					Message:  err.Error(),
					Fix: &Fix{
						Description: "Install the Remoteproc Runtime",
						Command:     fmt.Sprintf("topo install remoteproc-runtime --target %s", target),
					},
				}}
			}
			return DependencyCheckResult{SuccessValue: "remoteproc-runtime"}
		},
	}
}

func NewDependencyOnRemoteprocRuntimeShim(target ssh.Destination, r runner.Runner) Dependency {
	return Dependency{
		Label: "Remoteproc Shim",
		Check: func(ctx context.Context) DependencyCheckResult {
			if err := r.BinaryExists(ctx, "containerd-shim-remoteproc-v1"); err != nil {
				return DependencyCheckResult{Failure: &DependencyCheckFailure{
					Severity: SeverityWarning,
					Message:  err.Error(),
					Fix: &Fix{
						Description: "Install the Remoteproc Runtime",
						Command:     fmt.Sprintf("topo install remoteproc-runtime --target %s", target),
					},
				}}
			}
			return DependencyCheckResult{SuccessValue: "containerd-shim-remoteproc-v1"}
		},
	}
}

func NewDependencyOnLscpu(r runner.Runner) Dependency {
	return Dependency{
		Label: "Hardware Info",
		Check: func(ctx context.Context) DependencyCheckResult {
			if err := r.BinaryExists(ctx, "lscpu"); err != nil {
				return DependencyCheckResult{Failure: &DependencyCheckFailure{Severity: SeverityError, Message: err.Error()}}
			}
			return DependencyCheckResult{SuccessValue: "lscpu"}
		},
	}
}
