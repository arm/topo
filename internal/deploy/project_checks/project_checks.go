package checks

import (
	"errors"
	"fmt"
	"strings"

	"github.com/arm/topo/internal/compose"
	"github.com/arm/topo/internal/output/logger"
	"github.com/arm/topo/internal/ssh"
	targetpkg "github.com/arm/topo/internal/target"
)

const linuxArm64Platform = "linux/arm64"

func isPlatformMissing(platform string) bool {
	return platform == ""
}

func isPlatformMismatch(platform string) bool {
	return !strings.HasPrefix(platform, linuxArm64Platform)
}

func EnsureProjectIsLinuxArm64Ready(composePath string) error {
	project, err := compose.ReadProject(composePath)
	if err != nil {
		return fmt.Errorf("failed to load compose project: %w", err)
	}

	serviceNames := project.ServiceNames()
	builder := strings.Builder{}

	for _, svcName := range serviceNames {
		svc := project.Services[svcName]

		runtime := strings.ToLower(strings.TrimSpace(svc.Runtime))
		if runtime != "" && strings.Contains(runtime, "remoteproc") {
			continue
		}

		if isPlatformMissing(svc.Platform) {
			builder.WriteString(fmt.Sprintf("- service %q is missing platform declaration (set platform: %s or configure remoteproc)\n", svcName, linuxArm64Platform))
		} else if isPlatformMismatch(svc.Platform) {
			builder.WriteString(fmt.Sprintf("- service %q declares platform %q (expected %s)\n", svcName, svc.Platform, linuxArm64Platform))
		}
	}

	if builder.Len() > 0 {
		return errors.New("project is not ready for linux/arm64 deployments:\n" + builder.String())
	}

	return nil
}

func EnsureSSHGatewayPortsAreDisabled(target ssh.Host, connectionOption targetpkg.ConnectionOptions) []logger.Entry {
	logs := []logger.Entry{}

	if target.IsPlainLocalhost() {
		return logs
	}

	conn := targetpkg.NewConnection(string(target), connectionOption)
	server := detectSSHServer(conn)
	switch server {
	case sshServerOpenSSH:
		logs = gatewayPortsCheckInOpenSSH(conn, target, logs)
	case sshServerDropbear:
		logs = gatewayPortsCheckInDropbear(conn, target, logs)
	default:
		logs = append(logs, logger.Entry{
			Level:   logger.Warning,
			Message: fmt.Sprintf("SSH GatewayPorts check skipped on %s: unable to detect SSH server type", target),
		})
	}
	return logs
}

func gatewayPortsCheckInDropbear(conn targetpkg.Connection, target ssh.Host, logs []logger.Entry) []logger.Entry {
	output, err := conn.Run("ps aux | grep dropbear")
	if err != nil {
		logs = append(logs, logger.Entry{
			Level:   logger.Warning,
			Message: fmt.Sprintf("SSH GatewayPorts check skipped on %s: failed to execute dropbear command: %v", target, err),
		})
		return logs
	}

	if strings.Contains(output, "-a") {
		logs = append(logs, logger.Entry{
			Level:   logger.Err,
			Message: fmt.Sprintf("SSH GatewayPorts must be disabled on %s: Dropbear detected with -a option", target),
		})
	}
	return logs
}

func gatewayPortsCheckInOpenSSH(conn targetpkg.Connection, target ssh.Host, logs []logger.Entry) []logger.Entry {
	output, err := conn.Run("sshd -T")
	if err == nil {
		gatewayPorts, ok := parseGatewayPortsFromSSHD(output)
		if !ok {
			logs = append(logs, logger.Entry{
				Level:   logger.Err,
				Message: fmt.Sprintf("SSH GatewayPorts setting not found in sshd -T output on %s; unable to confirm it's disabled. Output may be incomplete or SSH version may not support this check", target),
			})
		} else if isGatewayPortsEnabled(gatewayPorts) {
			logs = append(logs, logger.Entry{
				Level:   logger.Err,
				Message: fmt.Sprintf("SSH GatewayPorts must be disabled on %s: expected \"no\", got %q. Update sshd_config and restart sshd", target, gatewayPorts),
			})
		}

		return logs
	} else {
		output, err = conn.Run("cat /etc/ssh/sshd_config")
		if err != nil {
			logs = append(logs, logger.Entry{
				Level:   logger.Err,
				Message: fmt.Sprintf("failed to read sshd_config on %s; unable to confirm SSH GatewayPorts is disabled. Error: %v", target, err),
			})
			return logs
		}

		gatewayPorts, uncertain := parseGatewayPortsFromConfig(output)
		if gatewayPorts == "" {
			logs = append(logs, logger.Entry{
				Level:   logger.Warning,
				Message: fmt.Sprintf("failed to determine SSH GatewayPorts setting on %s: not configured in sshd_config", target),
			})
		} else if isGatewayPortsEnabled(gatewayPorts) {
			logs = append(logs, logger.Entry{
				Level:   logger.Err,
				Message: fmt.Sprintf("SSH GatewayPorts must be disabled on %s: expected \"no\", got %q. Update sshd_config and restart sshd", target, gatewayPorts),
			})
		} else if uncertain {
			logs = append(logs, logger.Entry{
				Level:   logger.Warning,
				Message: fmt.Sprintf("SSH GatewayPorts setting on %s is uncertain due to Match or Include directives in sshd_config", target),
			})
		}
		return logs
	}
}

type sshServerType string

const (
	sshServerUnknown  sshServerType = "unknown"
	sshServerOpenSSH  sshServerType = "openssh"
	sshServerDropbear sshServerType = "dropbear"
)

func detectSSHServer(conn targetpkg.Connection) sshServerType {
	output, err := conn.Run("ps -eo comm")
	if err != nil {
		return sshServerUnknown
	}

	lines := strings.Split(output, "\n")
	hasOpenSSH := false
	hasDropbear := false
	for _, line := range lines {
		switch strings.TrimSpace(line) {
		case "sshd":
			hasOpenSSH = true
		case "dropbear":
			hasDropbear = true
		}
	}

	if hasOpenSSH {
		return sshServerOpenSSH
	}
	if hasDropbear {
		return sshServerDropbear
	}
	return sshServerUnknown
}

func isGatewayPortsEnabled(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "yes", "clientspecified":
		return true
	default:
		return false
	}
}

func parseGatewayPortsFromSSHD(sshdConfig string) (string, bool) {
	lines := strings.Split(sshdConfig, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		fields := strings.Fields(lines[i])
		if len(fields) < 2 {
			continue
		}
		if strings.EqualFold(fields[0], "gatewayports") {
			return fields[1], true
		}
	}
	return "", false
}

func parseGatewayPortsFromConfig(sshdConfig string) (value string, uncertain bool) {
	lines := strings.Split(sshdConfig, "\n")
	value = ""
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		if strings.EqualFold(fields[0], "match") || strings.EqualFold(fields[0], "include") {
			uncertain = true
		}
		if strings.EqualFold(fields[0], "gatewayports") {
			value = fields[1]
		}
	}
	return value, uncertain
}
