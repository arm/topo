package probe

import (
	"context"
	"errors"
	"strings"

	"github.com/arm/topo/internal/runner"
)

type RemoteProc struct {
	Name string `yaml:"name" json:"name"`
}

func Remoteprocs(ctx context.Context, r runner.Runner) ([]RemoteProc, error) {
	var remoteProcs []RemoteProc
	out, err := r.Run(ctx, "cat /sys/class/remoteproc/*/name")
	if err != nil {
		if errors.Is(err, runner.ErrTimeout) {
			return remoteProcs, err
		}
		return remoteProcs, nil
	}

	procs := strings.FieldsSeq(out)
	for proc := range procs {
		remoteProcs = append(remoteProcs, RemoteProc{Name: proc})
	}
	return remoteProcs, nil
}
