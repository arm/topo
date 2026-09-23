package health

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
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

func NewDependencyOnDockerCLI(r runner.Runner) Dependency {
	return Dependency{
		Label: "Docker CLI",
		Check: func(ctx context.Context) DependencyCheckResult {
			if err := r.BinaryExists(ctx, "docker"); err != nil {
				return DependencyCheckResult{Failure: &DependencyCheckFailure{
					Severity: SeverityError,
					Message:  err.Error(),
					Fix:      &Fix{Description: "Install a supported container engine. See " + containerEngineInstallURL},
				}}
			}
			return DependencyCheckResult{SuccessValue: "docker"}
		},
	}
}

func NewDependencyOnDockerDaemon(r runner.Runner) Dependency {
	return Dependency{
		Label: "Docker daemon",
		Check: func(ctx context.Context) DependencyCheckResult {
			if _, _, err := r.Run(ctx, "docker info"); err != nil {
				return DependencyCheckResult{Failure: &DependencyCheckFailure{
					Severity: SeverityError,
					Message:  err.Error(),
					Fix:      &Fix{Description: "Ensure Docker is running and the current user can run docker commands. See " + containerEngineInstallURL},
				}}
			}
			return DependencyCheckResult{SuccessValue: "reachable"}
		},
	}
}

func NewDependencyOnRemoteDockerDaemon(target ssh.Destination, probeInfo func(context.Context, ssh.Destination) error) Dependency {
	return Dependency{
		Label: "Docker daemon",
		Check: func(ctx context.Context) DependencyCheckResult {
			if err := probeInfo(ctx, target); err != nil {
				return DependencyCheckResult{Failure: &DependencyCheckFailure{
					Severity: SeverityError,
					Message:  err.Error(),
					Fix:      &Fix{Description: "Ensure docker is installed and running on the target. See " + containerEngineInstallURL},
				}}
			}
			return DependencyCheckResult{SuccessValue: "reachable"}
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

type ConnectivityOperations struct {
	Authenticate    func(context.Context, ssh.Destination) error
	KnownHostsEntry func(ssh.Destination) (string, error)
}

func NewConnectivityDependency(target ssh.Destination, operations ConnectivityOperations) Dependency {
	return Dependency{
		ID:    DependencyIDConnectivity,
		Label: "Connectivity",
		Check: func(ctx context.Context) DependencyCheckResult {
			err := operations.Authenticate(ctx, target)
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
				knownHostsEntry, configErr := operations.KnownHostsEntry(target)
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

func NewDependencyOnPodmanCLI(r runner.Runner) Dependency {
	return Dependency{
		Label: "Podman CLI",
		Check: func(ctx context.Context) DependencyCheckResult {
			if err := r.BinaryExists(ctx, "podman"); err != nil {
				return DependencyCheckResult{Failure: &DependencyCheckFailure{
					Severity: SeverityError,
					Message:  err.Error(),
					Fix:      &Fix{Description: "Install Podman. See " + containerEngineInstallURL},
				}}
			}
			return DependencyCheckResult{SuccessValue: "podman"}
		},
	}
}

func NewDependencyOnPodmanConnection(r runner.Runner) Dependency {
	return Dependency{
		Label: "Podman connection",
		Check: func(ctx context.Context) DependencyCheckResult {
			if _, _, err := r.Run(ctx, "podman info"); err != nil {
				return DependencyCheckResult{Failure: &DependencyCheckFailure{
					Severity: SeverityError,
					Message:  err.Error(),
					Fix:      &Fix{Description: "Start a Podman machine or configure an active Podman system connection, then ensure the current user can run Podman commands. See " + containerEngineInstallURL},
				}}
			}
			return DependencyCheckResult{SuccessValue: "reachable"}
		},
	}
}

func NewDependencyOnDockerComposeForPodman(composeVersion func(context.Context) error) Dependency {
	return Dependency{
		Label: "Podman Compose",
		Check: func(ctx context.Context) DependencyCheckResult {
			if err := composeVersion(ctx); err != nil {
				return DependencyCheckResult{Failure: &DependencyCheckFailure{
					Severity: SeverityError,
					Message:  err.Error(),
					Fix:      &Fix{Description: "Ensure docker-compose is on the $PATH. See " + containerEngineInstallURL},
				}}
			}
			return DependencyCheckResult{SuccessValue: "docker-compose"}
		},
	}
}

func NewDependencyOnRemotePodmanAPI(check func(context.Context) probe.RemotePodmanProbeResult) Dependency {
	return Dependency{
		Label: "Podman API",
		Check: func(ctx context.Context) DependencyCheckResult {
			result := check(ctx)
			if result.Err == nil {
				return DependencyCheckResult{SuccessValue: result.SocketPath}
			}
			switch {
			case errors.Is(result.Err, context.DeadlineExceeded):
				return DependencyCheckResult{Failure: &DependencyCheckFailure{
					Severity: SeverityError,
					Message:  "health check timed out",
					Fix:      &Fix{Description: "Retry the health check with a longer timeout."},
				}}
			case errors.Is(result.Err, probe.ErrRemotePodmanSocketResolutionFailed):
				return remotePodmanFailure(
					result.Err,
					"could not discover the Podman API socket on the target",
					&Fix{Description: "Start the Podman API socket and ensure the SSH user can access it. See " + containerEngineInstallURL},
				)
			case errors.Is(result.Err, probe.ErrRemotePodmanForwardingFailed):
				return remotePodmanFailure(
					result.Err,
					"could not open topo’s temporary SSH tunnel to the target Podman API",
					&Fix{Description: fmt.Sprintf("Ensure the target SSH server permits local TCP forwarding to the target Podman API socket at %s.", result.SocketPath)},
				)
			case errors.Is(result.Err, probe.ErrRemotePodmanAPIRequestFailed):
				return remotePodmanFailure(
					result.Err,
					"host-side Podman could not query the target API through topo’s temporary SSH tunnel",
					&Fix{Description: fmt.Sprintf("Ensure the Podman API socket at %s is functional and accessible to the SSH user.", result.SocketPath)},
				)
			case errors.Is(result.Err, probe.ErrRemotePodmanSocketTunnelCloseFailed):
				return remotePodmanFailure(
					result.Err,
					"could not close topo’s temporary SSH tunnel to the target Podman API",
					nil,
				)
			default:
				return remotePodmanFailure(
					result.Err,
					"",
					nil,
				)
			}
		},
	}
}

// remotePodmanFailure hides exit statuses from topo's internal commands. The
// commands use a temporary SSH tunnel and environment, so users cannot rerun
// them as shown to investigate the failure.
func remotePodmanFailure(err error, messageUnlessActionableError string, fix *Fix) DependencyCheckResult {
	message := err.Error()
	if _, ok := errors.AsType[*exec.ExitError](err); ok && messageUnlessActionableError != "" {
		message = messageUnlessActionableError
	}
	return DependencyCheckResult{Failure: &DependencyCheckFailure{
		Severity: SeverityError,
		Message:  message,
		Fix:      fix,
	}}
}
