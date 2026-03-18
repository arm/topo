package ssh

import (
	"fmt"
	"net"
	"strings"
	"unicode"
)

type SSHDestination string

const PlainLocalhost = SSHDestination("localhost")

func (h SSHDestination) IsPlainLocalhost() bool {
	return strings.EqualFold(string(h), "localhost") || h == "127.0.0.1"
}

func (h SSHDestination) IsLocalhost() bool {
	if h.IsPlainLocalhost() {
		return true
	}
	_, host, _ := SplitUserHostPort(string(h))
	return SSHDestination(host).IsPlainLocalhost()
}

func (h SSHDestination) AsURI() string {
	const scheme = "ssh://"
	withoutScheme := strings.TrimPrefix(string(h), scheme)
	return fmt.Sprintf("ssh://%s", withoutScheme)
}

func (h SSHDestination) Slugify() string {
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
