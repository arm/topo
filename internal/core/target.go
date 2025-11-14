package core

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"github.com/arm-debug/topo-cli/internal/dependencies"
)

type execSSH func(target, command string) (string, error)

type Target struct {
	SshConn         string
	ConnectionError error
	Features        []string
	Dependencies    []dependencies.Status
	RemoteCPU       []string
	exec            execSSH
}

func MakeTarget(sshTarget string, exec execSSH) Target {
	target := Target{}
	target.SshConn = sshTarget
	target.exec = exec
	_, err := target.Run("")
	if err != nil {
		target.ConnectionError = err
		return target
	}

	target.collectFeatures()
	target.Dependencies = dependencies.Check(dependencies.TargetRequiredDependencies, target.BinaryExists)
	target.collectRemoteCPU()

	return target
}

func (t *Target) Run(command string) (string, error) {
	return t.exec(t.SshConn, command)
}

func (t *Target) BinaryExists(bin string) (bool, error) {
	if !dependencies.BinaryRegex.MatchString(bin) {
		return false, fmt.Errorf("%q is not a valid binary name (contains invalid characters)", bin)
	}
	_, err := t.exec(t.SshConn, fmt.Sprintf("command -v %s", bin))
	return err == nil, nil
}

func (t *Target) collectFeatures() error {
	out, err := t.Run("grep -m1 Features /proc/cpuinfo")
	if err != nil {
		return err
	}
	t.Features = strings.Fields(out)

	if len(t.Features) > 0 && t.Features[0] == "Features:" {
		t.Features = t.Features[1:]
	}
	return nil
}

func (t *Target) collectRemoteCPU() error {
	out, err := t.Run("ls /sys/class/remoteproc")
	if err != nil {
		return err
	}

	if out == "" {
		return fmt.Errorf("target supports remoteproc, but no processors found")
	}

	out, err = t.Run("cat /sys/class/remoteproc/*/name")
	if err != nil {
		return err
	}

	t.RemoteCPU = strings.Fields(out)
	return nil
}

func ExecSSH(target, command string) (string, error) {
	cmd := exec.Command("ssh", target, command)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("ssh command to %s failed: %w | stderr: %s", target, err, stderr.String())
	}

	return stdout.String(), nil
}
