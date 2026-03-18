package ssh

import (
	"fmt"
	"net"
	"strings"
	"unicode"
)

type Destination string

const PlainLocalhost = Destination("localhost")

func (h Destination) IsPlainLocalhost() bool {
	return strings.EqualFold(string(h), "localhost") || h == "127.0.0.1"
}

func (h Destination) IsLocalhost() bool {
	if h.IsPlainLocalhost() {
		return true
	}
	_, host, _ := SplitUserHostPort(string(h))
	return Destination(host).IsPlainLocalhost()
}

func (h Destination) AsURI() string {
	const scheme = "ssh://"
	withoutScheme := strings.TrimPrefix(string(h), scheme)
	return fmt.Sprintf("ssh://%s", withoutScheme)
}

func (h Destination) Slugify() string {
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
