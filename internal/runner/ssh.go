package runner

import (
	"fmt"
	"runtime"
	"time"

	"github.com/arm/topo/internal/ssh"
)

type SSHOptions struct {
	Multiplex      bool
	ConnectTimeout time.Duration
}

func (opts SSHOptions) SSHArgs() []string {
	var args []string
	if opts.ConnectTimeout > 0 {
		args = append(args, "-o", fmt.Sprintf("ConnectTimeout=%d", int(opts.ConnectTimeout.Seconds())))
	}
	if opts.Multiplex && runtime.GOOS != "windows" {
		args = append(args, "-o", "ControlMaster=auto", "-o", "ControlPersist=10s", "-o", "ControlPath=~/.ssh/topo-cm-%r@%h:%p")
	}
	return args
}

type SSH struct {
	dest ssh.Destination
	opts SSHOptions
}

func NewSSH(dest ssh.Destination, opts SSHOptions) *SSH {
	opts.ConnectTimeout = ssh.NewConfig(dest).ConnectTimeout(opts.ConnectTimeout)
	return &SSH{dest: dest, opts: opts}
}

func (r *SSH) Run(cmdStr string) (string, error) {
	return r.exec(cmdStr, nil, nil)
}

func (r *SSH) RunWithStdin(cmdStr string, stdin []byte) (string, error) {
	return r.exec(cmdStr, stdin, nil)
}

func (r *SSH) RunWithArgs(cmdStr string, args ...string) (string, error) {
	args = append(args, r.opts.SSHArgs()...)
	return ssh.RunCommand(r.dest, cmdStr, nil, args...)
}

func (r *SSH) exec(cmdStr string, stdin []byte, extraSSHArgs []string) (string, error) {
	return ssh.RunCommand(r.dest, cmdStr, stdin, extraSSHArgs...)
}
