package probe

import (
	"context"
	"fmt"

	"github.com/arm/topo/internal/runner"
)

type HardwareProfile struct {
	HostProcessors   []HostProcessor `yaml:"hostProcessors" json:"hostProcessors"`
	RemoteProcessors []RemoteProc    `yaml:"remoteProcessors" json:"remoteProcessors,omitempty"`
	TotalMemoryKb    int64           `yaml:"totalMemoryKb" json:"totalMemoryKb"`
}

func Hardware(ctx context.Context, r runner.Runner) (HardwareProfile, error) {
	var hp HardwareProfile

	hostProcessors, err := HostProcessors(ctx, r)
	if err != nil {
		return hp, fmt.Errorf("collecting CPU info: %w", err)
	}
	hp.HostProcessors = hostProcessors

	remoteProcessors, err := Remoteprocs(ctx, r)
	if err != nil {
		return hp, fmt.Errorf("collecting remote processors: %w", err)
	}
	hp.RemoteProcessors = remoteProcessors

	memTotal, err := Memory(ctx, r)
	if err != nil {
		return hp, fmt.Errorf("collecting memory info: %w", err)
	}
	hp.TotalMemoryKb = memTotal

	return hp, nil
}
