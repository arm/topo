package netprobe

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"strings"
)

// IsRemotePortListening reports whether host:port accepts TCP connections.
func IsRemotePortListening(host, port string) (bool, error) {
	address := net.JoinHostPort(host, port)
	var remoteIP strings.Builder
	// Use curl so host resolution matches OpenSSH for mDNS, NSS, and split-DNS configurations.
	// - remote_ip detects a connection even when HTTP fails
	// - noproxy ensures the connection is direct
	// - silent suppresses curl's progress and errors.
	cmd := exec.Command("curl", "http://"+address, "--max-time", "5", "--noproxy", "*", "--output", os.DevNull, "--silent", "--write-out", "%{remote_ip}")
	cmd.Stdout = &remoteIP
	cmd.Stderr = io.Discard
	err := cmd.Run()
	if err == nil || remoteIP.Len() > 0 {
		return true, nil
	}

	exitError, ok := errors.AsType[*exec.ExitError](err)
	if ok {
		switch exitError.ExitCode() {
		case 7:
			return false, nil
		case 28:
			return false, fmt.Errorf("timed out while checking whether remote port %s is exposed: %w", address, err)
		case 6:
			return false, fmt.Errorf("could not resolve remote host %q while checking tunnel exposure: %w", host, err)
		}
	}

	return false, fmt.Errorf("could not verify whether remote port %s is exposed: %w", address, err)
}
