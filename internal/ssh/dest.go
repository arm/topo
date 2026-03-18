package ssh

import (
	"fmt"
	"net"
	"strings"
	"unicode"
)

type Destination string

const PlainLocalhost = Destination("localhost")

func (d Destination) IsPlainLocalhost() bool {
	return strings.EqualFold(string(d), "localhost") || d == "127.0.0.1"
}

func (d Destination) IsLocalhost() bool {
	if d.IsPlainLocalhost() {
		return true
	}
	_, host, _ := SplitUserHostPort(string(d))
	return Destination(host).IsPlainLocalhost()
}

func (d Destination) AsURI() string {
	const scheme = "ssh://"
	withoutScheme := strings.TrimPrefix(string(d), scheme)
	return fmt.Sprintf("ssh://%s", withoutScheme)
}

func (d Destination) Slugify() string {
	var b strings.Builder
	for _, r := range d {
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
