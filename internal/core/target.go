package core

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"github.com/arm-debug/topo-cli/internal/dependencies"
)

type execSSH func(target, command string) (string, error)

type HardwareProfile struct {
	Features  []string
	RemoteCPU []string
}

type TargetStatus struct {
	SSHTarget       string
	ConnectionError error
	Dependencies    []dependencies.Status
	Hardware        HardwareProfile
}

type TargetConnection struct {
	sshTarget string
	exec      execSSH
}

func NewTargetConnection(sshTarget string, exec execSSH) TargetConnection {
	return TargetConnection{
		sshTarget: sshTarget,
		exec:      exec,
	}
}

func (t *TargetConnection) Run(command string) (string, error) {
	return t.exec(t.sshTarget, command)
}

func (t *TargetConnection) BinaryExists(bin string) (bool, error) {
	if !dependencies.BinaryRegex.MatchString(bin) {
		return false, fmt.Errorf("%q is not a valid binary name (contains invalid characters)", bin)
	}
	_, err := t.exec(t.sshTarget, fmt.Sprintf("command -v %s", bin))
	return err == nil, nil
}

func (t *TargetConnection) Probe() TargetStatus {
	var targetStatus TargetStatus
	targetStatus.SSHTarget = t.sshTarget

	if err := t.ProbeConnection(); err != nil {
		targetStatus.ConnectionError = err
		return targetStatus
	}

	targetStatus.Dependencies = t.CheckDependencies()
	targetStatus.Hardware, _ = t.ProbeHardware()

	return targetStatus
}

func (t *TargetConnection) ProbeConnection() error {
	_, err := t.Run("true")
	return err
}

func (t *TargetConnection) CheckDependencies() []dependencies.Status {
	return dependencies.Check(dependencies.TargetRequiredDependencies, t.BinaryExists)
}

func (t *TargetConnection) ProbeHardware() (HardwareProfile, error) {
	var hp HardwareProfile

	if feats, err := t.collectFeatures(); err == nil {
		hp.Features = feats
	}
	if cpus, err := t.collectRemoteCPU(); err == nil {
		hp.RemoteCPU = cpus
	}

	return hp, nil
}

func (t *TargetConnection) collectFeatures() ([]string, error) {
	out, err := t.Run("grep -m1 Features /proc/cpuinfo")
	if err != nil {
		return nil, err
	}
	features := strings.Fields(out)

	if len(features) > 0 && features[0] == "Features:" {
		features = features[1:]
	}
	return features, nil
}

func (t *TargetConnection) collectRemoteCPU() ([]string, error) {
	out, err := t.Run("ls /sys/class/remoteproc")
	if err != nil {
		return nil, err
	}

	if out == "" {
		return nil, fmt.Errorf("target supports remoteproc, but no processors found")
	}

	out, err = t.Run("cat /sys/class/remoteproc/*/name")
	if err != nil {
		return nil, err
	}

	remoteCPU := strings.Fields(out)
	return remoteCPU, nil
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
