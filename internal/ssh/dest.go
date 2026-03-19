package ssh

import (
	"fmt"
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
	config := NewConfig(string(d))
	return Destination(config.Host).IsPlainLocalhost()
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
