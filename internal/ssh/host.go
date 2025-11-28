package ssh

import (
	"fmt"
	"strings"
)

type Host string

const PlainLocalhost = Host("localhost")

func (h Host) IsPlainLocalhost() bool {
	return strings.EqualFold(string(h), "localhost") || h == "127.0.0.1"
}

func (h Host) AsURI() string {
	return fmt.Sprintf("ssh://%s", h)
}
