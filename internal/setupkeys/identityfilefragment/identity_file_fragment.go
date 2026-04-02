package identityfilefragment

import (
	"io"

	"github.com/arm/topo/internal/ssh"
)

type IdentityFileFragmentWrite struct {
	privateKeyPath string
	targetSlug     string
	destination    ssh.Destination
}

func NewIdentityFileFragmentWrite(privKeyPath string, targetSlug string, destination ssh.Destination) *IdentityFileFragmentWrite {
	return &IdentityFileFragmentWrite{
		privateKeyPath: privKeyPath,
		targetSlug:     targetSlug,
		destination:    destination,
	}
}

func (ifw *IdentityFileFragmentWrite) Description() string {
	return "Write IdentityFile fragment to SSH config for target"
}

func (ifw *IdentityFileFragmentWrite) Run(outputWriter io.Writer) error {
	directives := []ssh.ConfigDirective{
		ssh.NewConfigDirectiveIdentityFile(ifw.privateKeyPath),
		ssh.NewDirective("IdentitiesOnly", "yes"),
	}

	return ssh.CreateOrModifyConfigFile(ifw.destination, ifw.targetSlug, directives)
}
