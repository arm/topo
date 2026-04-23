package probe

import (
	"context"
	"errors"
	"strings"

	"github.com/arm/topo/internal/runner"
)

type RemoteprocCPU struct {
	Name string `yaml:"name" json:"name"`
}

func Remoteproc(ctx context.Context, r runner.Runner) ([]RemoteprocCPU, error) {
	var remoteProcs []RemoteprocCPU
	out, err := r.Run(ctx, "cat /sys/class/remoteproc/*/name")
	if err != nil {
		if errors.Is(err, runner.ErrTimeout) {
			return remoteProcs, err
		}
		return remoteProcs, nil
	}

	remoteCPU := strings.FieldsSeq(out)
	for cpu := range remoteCPU {
		remoteProcs = append(remoteProcs, RemoteprocCPU{Name: cpu})
	}
	return remoteProcs, nil
}
