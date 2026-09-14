package health

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/arm/topo/internal/command"
	"github.com/arm/topo/internal/probe"
	"github.com/arm/topo/internal/runner"
	"github.com/arm/topo/internal/ssh"
)

const containerEngineInstallURL = "https://github.com/arm/topo#install-a-container-engine"

type DependencyID string

const (
	DependencyIDConnectivity DependencyID = "target-connectivity"
	DependencyIDRemoteproc   DependencyID = "remoteproc"
)

type Dependency struct {
	ID            DependencyID
	Label         string
	Check         DependencyCheckFn
	Prerequisites []DependencyID
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

func HostRequiredDependencies(skipVersionChecks bool) []Dependency {
	r := runner.NewLocal()

	topo := Dependency{
		ID:    DependencyID("topo"),
		Label: "Topo",
		Check: func(ctx context.Context) DependencyCheckResult {
			if skipVersionChecks {
				return DependencyCheckResult{SuccessValue: "topo"}
			}
			if failure := CheckTopoIsUpToDate(ctx); failure != nil {
				return DependencyCheckResult{Failure: failure}
			}
			return DependencyCheckResult{SuccessValue: "topo"}
		},
	}

	ssh := Dependency{
		ID:    DependencyID("ssh"),
		Label: "OpenSSH",
		Check: func(ctx context.Context) DependencyCheckResult {
			if err := r.BinaryExists(ctx, "ssh"); err != nil {
				return DependencyCheckResult{Failure: &DependencyCheckFailure{Severity: SeverityError, Message: err.Error()}}
			}
			if failure := CheckOpenSSHAvailable(ctx, r, "ssh"); failure != nil {
				return DependencyCheckResult{Failure: failure}
			}
			return DependencyCheckResult{SuccessValue: "ssh"}
		},
	}

	docker := Dependency{
		ID:    DependencyID("host-docker"),
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

	dockerCompose := Dependency{
		ID:    DependencyID("docker-compose"),
		Label: "Docker Compose",
		Check: func(ctx context.Context) DependencyCheckResult {
			if _, _, err := r.Run(ctx, "docker-compose"); err != nil {
				return DependencyCheckResult{Failure: &DependencyCheckFailure{
					Severity: SeverityError,
					Message:  err.Error(),
					Fix:      &Fix{Description: "Ensure Docker Compose is installed as a plugin for Docker. See " + containerEngineInstallURL},
				}}
			}
			if failure := CheckDockerComposeMinVersion(ctx, r, "2.21.0"); failure != nil {
				return DependencyCheckResult{Failure: failure}
			}
			return DependencyCheckResult{SuccessValue: "docker-compose"}
		},
		Prerequisites: []DependencyID{docker.ID},
	}

	return []Dependency{topo, ssh, docker, dockerCompose}
}

func TargetRequiredDependencies(target ssh.Destination, acceptNewHostKeys bool) []Dependency {
	r := runner.For(target)

	remoteTargetPrerequisites := []DependencyID(nil)
	dependencies := []Dependency(nil)
	if !target.IsPlainLocalhost() {
		connectivity := NewConnectivityDependency(target, acceptNewHostKeys)
		dependencies = append(dependencies, connectivity)
		remoteTargetPrerequisites = []DependencyID{connectivity.ID}
	}

	docker := Dependency{
		ID:            DependencyID("target-docker"),
		Label:         "Container Engine",
		Prerequisites: remoteTargetPrerequisites,
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

	remoteproc := NewRemoteprocDependency(r, remoteTargetPrerequisites...)

	remoteprocRuntime := NewRemoteprocRuntimeDependency(
		target,
		r,
		append([]DependencyID{docker.ID, remoteproc.ID}, remoteTargetPrerequisites...)...,
	)

	remoteprocRuntimeShim := Dependency{
		ID:            DependencyID("containerd-shim-remoteproc-v1"),
		Label:         "Remoteproc Shim",
		Prerequisites: append([]DependencyID{docker.ID, remoteproc.ID}, remoteTargetPrerequisites...),
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

	lscpu := Dependency{
		ID:            DependencyID("lscpu"),
		Label:         "Hardware Info",
		Prerequisites: remoteTargetPrerequisites,
		Check: func(ctx context.Context) DependencyCheckResult {
			if err := r.BinaryExists(ctx, "lscpu"); err != nil {
				return DependencyCheckResult{Failure: &DependencyCheckFailure{Severity: SeverityError, Message: err.Error()}}
			}
			return DependencyCheckResult{SuccessValue: "lscpu"}
		},
	}

	return append(dependencies, docker, remoteproc, remoteprocRuntime, remoteprocRuntimeShim, lscpu)
}

func NewConnectivityDependency(target ssh.Destination, acceptNewHostKeys bool) Dependency {
	sshRunner := runner.NewSSH(target)
	return Dependency{
		ID:    DependencyIDConnectivity,
		Label: "Connectivity",
		Check: func(ctx context.Context) DependencyCheckResult {
			err := probe.SSHAuthentication(ctx, sshRunner, acceptNewHostKeys)
			if err == nil {
				return DependencyCheckResult{}
			}

			failure := DependencyCheckFailure{Severity: SeverityError, Message: err.Error()}
			switch {
			case errors.Is(err, probe.ErrAuthFailed), errors.Is(err, probe.ErrTooManyAuthFails):
				failure.Fix = &Fix{
					Description: "Configure SSH keys on remote target",
					Command:     fmt.Sprintf("topo setup-keys --target %s", target),
				}
			case errors.Is(err, probe.ErrHostKeyUnknown):
				failure.Fix = &Fix{
					Description: "Trust the target's SSH host key",
					Command:     fmt.Sprintf("topo health --target %s --accept-new-host-keys", target),
				}
			case errors.Is(err, probe.ErrHostKeyChanged):
				sshConfig, configErr := ssh.LoadConfig(target)
				fixCommand := ""
				if configErr == nil {
					fixCommand = fmt.Sprintf("ssh-keygen -R %s", command.QuoteArg(sshConfig.AsKnownHostsEntry()))
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

func NewRemoteprocDependency(r runner.Runner, prerequisites ...DependencyID) Dependency {
	return Dependency{
		ID:            DependencyIDRemoteproc,
		Label:         "Processing Domain Driver (remoteproc)",
		Prerequisites: prerequisites,
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

func NewRemoteprocRuntimeDependency(target ssh.Destination, r runner.Runner, prerequisites ...DependencyID) Dependency {
	return Dependency{
		ID:            DependencyID("remoteproc-runtime"),
		Label:         "Remoteproc Runtime",
		Prerequisites: prerequisites,
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

type DependencyStatus struct {
	Dependency Dependency
	Result     DependencyCheckResult
}

func PerformChecks(ctx context.Context, dependencies []Dependency) []DependencyStatus {
	healthy := make(map[DependencyID]struct{})
	result := make([]DependencyStatus, 0, len(dependencies))

	for _, dep := range dependencies {
		if !allPrerequisitesFulfilled(dep.Prerequisites, healthy) {
			continue
		}

		checkResult := DependencyCheckResult{}
		if dep.Check != nil {
			checkResult = dep.Check(ctx)
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
