package ssh

import "fmt"

type Host string

const Empty = Host("")

func (h Host) AsURI() string {
	return fmt.Sprintf("ssh://%s", h)
}
