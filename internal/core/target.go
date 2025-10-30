package core

import (
	"bytes"
	"fmt"
	"strings"
)

type execSSH func(target, command string) (string, error)

type Target struct {
	sshConn         string
	connectionError error
	features        []string
	exec            execSSH
}

func MakeTarget(sshTarget string, exec execSSH) Target {
	target := Target{}
	target.sshConn = sshTarget
	target.exec = exec
	_, err := target.Run("")
	if err != nil {
		target.connectionError = err
		return target
	}

	target.collectFeatures()
	return target
}

func (t *Target) Run(command string) (string, error) {
	return t.exec(t.sshConn, command)
}

func (t *Target) collectFeatures() error {
	out, err := t.Run("grep -m1 Features /proc/cpuinfo")
	if err != nil {
		return err
	}
	t.features = strings.Fields(out)
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
