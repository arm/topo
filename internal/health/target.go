package health

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

type execSSH func(target, command string) (string, error)

type HardwareProfile struct {
	Features  []string
	RemoteCPU []string
}

func (hw HardwareProfile) Capabilities() map[HardwareCapability]struct{} {
	capabilities := make(map[HardwareCapability]struct{})
	if len(hw.RemoteCPU) > 0 {
		capabilities[Remoteproc] = struct{}{}
	}
	return capabilities
}

type Status struct {
	SSHTarget       string
	ConnectionError error
	Dependencies    []DependencyStatus
	Hardware        HardwareProfile
}

type Connection struct {
	sshTarget string
	exec      execSSH
}

func NewConnection(sshTarget string, exec execSSH) Connection {
	return Connection{
		sshTarget: sshTarget,
		exec:      exec,
	}
}

func (c *Connection) Run(command string) (string, error) {
	return c.exec(c.sshTarget, command)
}

func (c *Connection) BinaryExists(bin string) (bool, error) {
	if !BinaryRegex.MatchString(bin) {
		return false, fmt.Errorf("%q is not a valid binary name (contains invalid characters)", bin)
	}
	_, err := c.exec(c.sshTarget, fmt.Sprintf("/bin/sh -c 'exec ${SHELL:-/bin/sh} -l -c \"command -v %s\"'", bin))
	return err == nil, nil
}

func (c *Connection) Probe() Status {
	var targetStatus Status
	targetStatus.SSHTarget = c.sshTarget

	if err := c.ProbeConnection(); err != nil {
		targetStatus.ConnectionError = err
		return targetStatus
	}

	targetStatus.Hardware, _ = c.ProbeHardware()
	targetStatus.Dependencies = c.CheckDependencies(targetStatus.Hardware.Capabilities())

	return targetStatus
}

func (c *Connection) ProbeConnection() error {
	_, err := c.Run("true")
	return err
}

func (c *Connection) CheckDependencies(hardware map[HardwareCapability]struct{}) []DependencyStatus {
	deps := FilterByHardware(TargetRequiredDependencies, hardware)
	return CheckInstalled(deps, c.BinaryExists)
}

func (c *Connection) ProbeHardware() (HardwareProfile, error) {
	var hp HardwareProfile

	if feats, err := c.collectFeatures(); err == nil {
		hp.Features = feats
	}
	if cpus, err := c.collectRemoteCPU(); err == nil {
		hp.RemoteCPU = cpus
	}

	return hp, nil
}

func (c *Connection) collectFeatures() ([]string, error) {
	out, err := c.Run("grep -m1 Features /proc/cpuinfo")
	if err != nil {
		return nil, err
	}
	features := strings.Fields(out)

	if len(features) > 0 && features[0] == "Features:" {
		features = features[1:]
	}
	return features, nil
}

func (c *Connection) collectRemoteCPU() ([]string, error) {
	out, err := c.Run("ls /sys/class/remoteproc")
	if err != nil {
		return nil, err
	}

	if out == "" {
		return nil, fmt.Errorf("target supports remoteproc, but no processors found")
	}

	out, err = c.Run("cat /sys/class/remoteproc/*/name")
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
