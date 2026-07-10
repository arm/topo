package probe

import (
	"context"
	"errors"
	"strings"

	"github.com/arm/topo/internal/runner"
)

type RemoteProcessor struct {
	Name string `yaml:"name" json:"name"`
}

func Remoteproc(ctx context.Context, r runner.Runner) ([]RemoteProcessor, error) {
	var remoteProcs []RemoteProcessor
	out, err := r.Run(ctx, "cat /sys/class/remoteproc/*/name")
	if err != nil {
		if errors.Is(err, runner.ErrTimeout) {
			return remoteProcs, err
		}
		return remoteProcs, nil
	}

	procs := strings.FieldsSeq(out)
	for proc := range procs {
		remoteProcs = append(remoteProcs, RemoteProcessor{Name: proc})
	}
	return remoteProcs, nil
}
