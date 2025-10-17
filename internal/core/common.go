package core

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/arm-debug/topo-cli/configs"
)

// Execution / logging seams (overridable in tests)
var ExecCommand = exec.Command
var LogPrintf = fmt.Printf

// Embedded version string (re-export for tests / external callers)
var VersionTxt = configs.VersionTxt

const TargetEnvVar = "TOPO_TARGET"

// Exported constants referenced externally
const (
	DefaultBoard           = "NXP i.MX 93"
	DefaultDockerContext   = "default"
	DefaultComposeFileName = "compose.topo.yaml"
)

// ResolveTarget returns the effective SSH target alias using precedence:
// 1) explicit flag value
// 2) TOPO_TARGET environment variable
// Errors if neither is provided.
func ResolveTarget(flagValue string) (string, error) {
	if strings.TrimSpace(flagValue) != "" {
		return flagValue, nil
	}
	if env := strings.TrimSpace(os.Getenv(TargetEnvVar)); env != "" {
		return env, nil
	}
	return "", fmt.Errorf("target not specified: provide --target or set TOPO_TARGET env var")
}

func getContextName(sshTarget string) string {
	return sshTarget
}
