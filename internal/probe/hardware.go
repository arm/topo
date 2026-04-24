package probe

import (
	"context"
	"fmt"

	"github.com/arm/topo/internal/runner"
)

type HardwareProfile struct {
	HostProcessors []HostProcessor `yaml:"hosts" json:"hosts"`
	RemoteCPUs     []RemoteprocCPU `yaml:"remoteprocs" json:"remoteprocs,omitempty"`
	TotalMemoryKb  int64           `yaml:"totalmemory_kb" json:"totalmemory_kb"`
}

func Hardware(ctx context.Context, r runner.Runner) (HardwareProfile, error) {
	var hp HardwareProfile

	hostProcessors, err := CPU(ctx, r)
	if err != nil {
		return hp, fmt.Errorf("collecting CPU info: %w", err)
	}
	hp.HostProcessors = hostProcessors

	remoteCPUs, err := Remoteproc(ctx, r)
	if err != nil {
		return hp, fmt.Errorf("collecting remote CPUs: %w", err)
	}
	hp.RemoteCPUs = remoteCPUs

	memTotal, err := Memory(ctx, r)
	if err != nil {
		return hp, fmt.Errorf("collecting memory info: %w", err)
	}
	hp.TotalMemoryKb = memTotal

	return hp, nil
}
