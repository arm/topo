package core

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/arm-debug/topo-cli/internal/dependencies"
)

type execSSH func(target, command string) (string, error)

type Target struct {
	SshConn         string
	ConnectionError error
	Features        []string
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

func ExecSSH(target, command string) (string, error) {
	cmd := ExecCommand("ssh", target, command)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("ssh command to %s failed: %w | stderr: %s", target, err, stderr.String())
	}

	return stdout.String(), nil
}
