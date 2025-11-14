package core

import (
	"fmt"
	"os/exec"
)

// Execution / logging seams (overridable in tests)
var (
	ExecCommand = exec.Command
	LogPrintf   = fmt.Printf
)

// Exported constants referenced externally
const (
	DefaultComposeFileName = "compose.yaml"
)
