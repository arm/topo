package ssh

import (
	"fmt"
	"net"
	"strings"
	"unicode"
)

type Destination struct {
	User string
	Host string
	Port string
}

func (d Destination) String() string {
	var builder strings.Builder
	fmt.Fprint(&builder, "ssh://")
	if d.User != "" {
		fmt.Fprint(&builder, d.User)
		fmt.Fprint(&builder, "@")
	}
	fmt.Fprint(&builder, d.Host)
	if d.Port != "" {
		fmt.Fprint(&builder, ":")
		fmt.Fprint(&builder, d.Port)
	}
	return builder.String()
}

func MustNewDestination(raw string) Destination {
	user, host, port := SplitUserHostPort(raw)
	return Destination{
		User: user,
		Host: host,
		Port: port,
	}
}

var PlainLocalhost = MustNewDestination("localhost")

func (d Destination) IsPlainLocalhost() bool {
	if d.Port != "" || d.User != "" {
		return false
	}
	return d.IsLocalhost()
}

func (d Destination) IsLocalhost() bool {
	return strings.EqualFold(d.Host, "localhost") || d.Host == "127.0.0.1"
}

func (d Destination) AsURI() string {
	return d.String()
}

func (d Destination) Slugify() string {
	var b strings.Builder
	uri := d.AsURI()
	withoutScheme := strings.TrimPrefix(uri, "ssh://")
	for _, r := range withoutScheme {
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
