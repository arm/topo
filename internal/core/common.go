package core

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/arm-debug/topo-cli/configs"
)

// Execution / logging seams (overridable in tests)
var ExecCommand = exec.Command
var LogPrintf = fmt.Printf

// Embedded version string (re-export for tests / external callers)
var VersionTxt = configs.VersionTxt

// Exported constants referenced externally
const (
	DefaultBoard           = "NXP i.MX 93"
	DefaultBoardHostname   = "topo.local"
	DefaultBoardUser       = "root"
	DefaultSshTarget       = DefaultBoardUser + "@" + DefaultBoardHostname
	DefaultSshUri          = "ssh://" + DefaultSshTarget
	DefaultDockerContext   = DefaultBoardHostname
	DefaultComposeFileName = "compose.topo.yaml"
)

// parseSshTarget splits user@host into components.
func parseSshTarget(sshTarget string) (string, string, error) {
	parts := strings.Split(sshTarget, "@")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid ssh target %q", sshTarget)
	}
	return parts[0], parts[1], nil
}
