package testutil

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

const vmName = "topo-test-docker"

var setupVMOnce = sync.OnceValues(setupVM)

type DockerVM struct {
	DockerSocketPath string
}

func RequireLima(t testing.TB) {
	t.Helper()
	if _, err := exec.LookPath("limactl"); err != nil {
		t.Skip("Lima not found. Install Lima: https://lima-vm.io/docs/installation/")
	}
}

func StartDockerVM(t *testing.T) *DockerVM {
	t.Helper()
	RequireLima(t)

	vm, err := setupVMOnce()
	if err != nil {
		t.Fatalf("failed to setup vm: %v", err)
	}

	return vm
}

type limaOperation int

const (
	limaOperationNone limaOperation = iota
	limaOperationCreate
	limaOperationStart
)

func setupVM() (*DockerVM, error) {
	operation, err := determineLimaOperation()
	if err != nil {
		return nil, err
	}

	if err := executeLimaOperation(operation); err != nil {
		return nil, err
	}

	socketPath, err := getDockerSocketPath()
	if err != nil {
		return nil, fmt.Errorf("failed to get Docker socket path: %w", err)
	}

	return &DockerVM{DockerSocketPath: socketPath}, nil
}

func determineLimaOperation() (limaOperation, error) {
	cmd := exec.Command("limactl", "list", "--format", "{{.Status}}", vmName)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() != 0 {
			return limaOperationCreate, nil
		}
		return limaOperationNone, fmt.Errorf("failed to get VM status: %w", err)
	}

	status := strings.TrimSpace(string(output))
	switch status {
	case "Running":
		return limaOperationNone, nil
	case "Stopped":
		return limaOperationStart, nil
	default:
		return limaOperationNone, fmt.Errorf("unexpected VM status: %s", status)
	}
}

func executeLimaOperation(operation limaOperation) error {
	var cmd *exec.Cmd
	switch operation {
	case limaOperationNone:
		return nil
	case limaOperationCreate:
		cmd = exec.Command("limactl", "start", "--name", vmName, "template://docker")
	case limaOperationStart:
		cmd = exec.Command("limactl", "start", vmName)
	default:
		return fmt.Errorf("unknown lima operation: %d", operation)
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to execute lima operation: %w", err)
	}
	return nil
}

func getDockerSocketPath() (string, error) {
	vmDir, err := getVMDirectory()
	if err != nil {
		return "", err
	}

	socketPath := filepath.Join(vmDir, "sock", "docker.sock")
	return "unix://" + socketPath, nil
}

func getVMDirectory() (string, error) {
	cmd := exec.Command("limactl", "list", vmName, "--format", "{{.Dir}}")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get VM directory: %w", err)
	}

	vmDir := strings.TrimSpace(string(output))
	if vmDir == "" {
		return "", fmt.Errorf("empty VM directory")
	}

	return vmDir, nil
}
