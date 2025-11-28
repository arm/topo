package testutil

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
)

type DockerVM struct {
	SSHConnectionString string
}

func StartDockerVM(t *testing.T) *DockerVM {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping test that requires a VM in short mode")
	}
	requireLima(t)

	vmName := generateVMName(t)

	t.Cleanup(func() {
		deleteVM(vmName)
	})

	unlock := acquireLimaLock(t)
	defer unlock()

	if err := createVM(vmName); err != nil {
		t.Fatalf("failed to create vm: %v", err)
	}

	sshConnection, err := getSSHConnectionString(vmName)
	if err != nil {
		t.Fatalf("failed to get SSH connection string: %v", err)
	}

	if err := ensureHostKeyKnown(sshConnection); err != nil {
		t.Fatalf("failed to add host key: %v", err)
	}

	return &DockerVM{SSHConnectionString: sshConnection}
}

func acquireLimaLock(t *testing.T) func() {
	t.Helper()
	lockPath := filepath.Join(os.TempDir(), "topo-lima-test.lock")
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		t.Fatalf("failed to open lock file: %v", err)
	}

	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		f.Close()
		t.Fatalf("failed to acquire lock: %v", err)
	}

	return func() {
		syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		f.Close()
	}
}

func requireLima(t testing.TB) {
	t.Helper()
	if _, err := exec.LookPath("limactl"); err != nil {
		t.Skip("Lima not found. Install Lima: https://lima-vm.io/docs/installation/")
	}
}

func generateVMName(t *testing.T) string {
	return fmt.Sprintf("topo-test-%s", SanitiseTestName(t))
}

func createVM(vmName string) error {
	deleteVM(vmName)
	templatePath := filepath.Join(getTestUtilDir(), "lima-template.yaml")
	cmd := exec.Command("limactl", "start", "--name", vmName, templatePath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create vm: %w", err)
	}
	return nil
}

func deleteVM(vmName string) {
	cmd := exec.Command("limactl", "delete", "--force", vmName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	_ = cmd.Run()
}

func getTestUtilDir() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Dir(filename)
}

func getSSHConnectionString(vmName string) (string, error) {
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

	_ = exec.Command("ssh-keygen", "-R", fmt.Sprintf("[%s]:%s", host, port)).Run()

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
