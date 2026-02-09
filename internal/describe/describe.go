package describe

import (
	"github.com/arm-debug/topo-cli/internal/health"
	"github.com/arm-debug/topo-cli/internal/ssh"
)

func Generate(sshTarget string) (health.HardwareProfile, error) {
	conn := health.NewConnection(sshTarget, ssh.ExecSSH)
	hwProfile, err := conn.ProbeHardware()
	if err != nil {
		return health.HardwareProfile{}, err
	}

	return hwProfile, nil
}
