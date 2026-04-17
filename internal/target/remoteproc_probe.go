package target

import (
	"context"
	"strings"

	"github.com/arm/topo/internal/runner"
)

type RemoteprocCPU struct {
	Name string `yaml:"name"`
}

func ProbeRemoteproc(ctx context.Context, r runner.Runner) ([]RemoteprocCPU, error) {
	var remoteProcs []RemoteprocCPU
	out, err := r.Run(ctx, "ls /sys/class/remoteproc")
	if err != nil || out == "" {
		return remoteProcs, nil
	}

	out, err = r.Run(ctx, "cat /sys/class/remoteproc/*/name")
	if err != nil {
		return remoteProcs, err
	}

	remoteCPU := strings.FieldsSeq(out)
	for cpu := range remoteCPU {
		remoteProcs = append(remoteProcs, RemoteprocCPU{Name: cpu})
	}
	return remoteProcs, nil
}
