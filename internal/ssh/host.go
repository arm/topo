package ssh

import (
	"crypto/sha256"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

type Host string

const PlainLocalhost = Host("localhost")

func (h Host) IsPlainLocalhost() bool {
	return strings.EqualFold(string(h), "localhost") || h == "127.0.0.1"
}

func (h Host) IsLocalhost() bool {
	if h.IsPlainLocalhost() {
		return true
	}
	_, host, _ := SplitUserHostPort(string(h))
	return Host(host).IsPlainLocalhost()
}

func (h Host) ControlSocketPath() string {
	hash := sha256.Sum256([]byte(h))
	hostHash := fmt.Sprintf("%x", hash[:8]) // Hash to avoid filepath limits
	return filepath.Join(os.TempDir(), fmt.Sprintf("topo-tunnel-%s", hostHash))
}

func (h Host) FormatSSHConnectCommand(useControlSockets bool, registryPort string) []string {
	args := []string{"ssh", "-N", "-o", "ExitOnForwardFailure=yes"}
	user, host, port := SplitUserHostPort(string(h))
	if port != "22" && port != "" {
		args = append(args, "-p", port)
	}
	if useControlSockets {
		args = append(args,
			"-fMS", h.ControlSocketPath(),
		)
	}
	args = append(args,
		"-R", fmt.Sprintf("%s:127.0.0.1:%s", registryPort, registryPort),
		formatSSHDestinationWithoutPort(user, host),
	)
	return args
}

func (h Host) GetHost() string {
	_, host, _ := SplitUserHostPort(string(h))
	return host
}

func formatSSHDestinationWithoutPort(user, host string) string {
	dest := host
	if strings.Contains(host, ":") {
		dest = "[" + host + "]"
	}
	if user != "" {
		dest = fmt.Sprintf("%s@%s", user, dest)
	}
	return dest
}

func (h Host) FormatSSHExitCommand() []string {
	args := []string{"ssh"}
	user, host, port := SplitUserHostPort(string(h))
	if port != "22" && port != "" {
		args = append(args, "-p", port)
	}
	args = append(args,
		"-S", h.ControlSocketPath(),
		"-O", "exit",
		formatSSHDestinationWithoutPort(user, host),
	)
	return args
}

func (h Host) AsURI() string {
	const scheme = "ssh://"
	withoutScheme := strings.TrimPrefix(string(h), scheme)
	return fmt.Sprintf("ssh://%s", withoutScheme)
}

func (h Host) Slugify() string {
	var b strings.Builder
	for _, r := range h {
		toWrite := '_'
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' || r == '.' {
			toWrite = r
		}
		b.WriteRune(toWrite)
	}
	return b.String()
}

func SplitUserHostPort(raw string) (user, host, port string) {
	hostPart := raw
	if at := strings.LastIndex(raw, "@"); at != -1 {
		user = raw[:at]
		hostPart = raw[at+1:]
	}

	if strings.HasPrefix(hostPart, "[") && strings.HasSuffix(hostPart, "]") {
		host = strings.TrimSuffix(strings.TrimPrefix(hostPart, "["), "]")
		return user, host, port
	}

	if h, p, err := net.SplitHostPort(hostPart); err == nil {
		return user, h, p
	}
	return user, hostPart, port
}
