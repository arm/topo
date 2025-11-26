package testutil

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

const vmName = "topo-test-docker"

var setupVMOnce = sync.OnceValues(setupVM)

type DockerVM struct {
	SSHConnectionString string
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

	sshConnection, err := getSSHConnectionString()
	if err != nil {
		return nil, fmt.Errorf("failed to get SSH connection string: %w", err)
	}

	if err := ensureHostKeyKnown(sshConnection); err != nil {
		return nil, fmt.Errorf("failed to add host key: %w", err)
	}

	return &DockerVM{SSHConnectionString: sshConnection}, nil
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
		templatePath := filepath.Join(getTestUtilDir(), "lima-template.yaml")
		cmd = exec.Command("limactl", "start", "--name", vmName, templatePath)
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

func getTestUtilDir() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Dir(filename)
}

func getSSHConnectionString() (string, error) {
	cmd := exec.Command("limactl", "list", vmName, "--format", "{{.SSHAddress}}:{{.SSHLocalPort}}")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get SSH connection string: %w", err)
	}

	sshConnection := strings.TrimSpace(string(output))
	if sshConnection == "" {
		return "", fmt.Errorf("empty SSH connection string")
	}

	return sshConnection, nil
}

func ensureHostKeyKnown(sshConnection string) error {
	parts := strings.Split(sshConnection, ":")
	if len(parts) != 2 {
		return fmt.Errorf("invalid SSH connection string format: %s", sshConnection)
	}

	host := parts[0]
	port := parts[1]

	cmd := exec.Command("ssh-keyscan", "-p", port, host)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("ssh-keyscan failed: %w", err)
	}

	if len(output) == 0 {
		return fmt.Errorf("ssh-keyscan returned no host keys")
	}

	knownHostsPath := filepath.Join(os.Getenv("HOME"), ".ssh", "known_hosts")
	if err := os.MkdirAll(filepath.Dir(knownHostsPath), 0700); err != nil {
		return fmt.Errorf("failed to create .ssh directory: %w", err)
	}

	f, err := os.OpenFile(knownHostsPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("failed to open known_hosts: %w", err)
	}
	defer f.Close()

	if _, err := f.Write(output); err != nil {
		return fmt.Errorf("failed to write to known_hosts: %w", err)
	}

	return nil
}
