package core

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Execution / logging seams (overridable in tests)
var (
	ExecCommand = exec.Command
	LogPrintf   = fmt.Printf
)

const TargetEnvVar = "TOPO_TARGET"

// Exported constants referenced externally
const (
	DefaultComposeFileName = "compose.yaml"
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
