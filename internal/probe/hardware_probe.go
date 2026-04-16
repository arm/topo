package probe

import (
	"context"
	"fmt"

	"github.com/arm/topo/internal/runner"
)

type HardwareProfile struct {
	HostProcessor []HostProcessor `yaml:"host" json:"host"`
	RemoteCPU     []RemoteprocCPU `yaml:"remoteprocs" json:"remoteprocs,omitempty"`
	TotalMemoryKb int64           `yaml:"totalmemory_kb" json:"totalmemory_kb"`
}

func ProbeHardware(ctx context.Context, r runner.Runner) (HardwareProfile, error) {
	var hp HardwareProfile

	cpuProfile, err := ProbeCPU(ctx, r)
	if err != nil {
		return hp, fmt.Errorf("collecting CPU info: %w", err)
	}
	hp.HostProcessor = cpuProfile

	cpus, err := ProbeRemoteproc(ctx, r)
	if err != nil {
		return hp, fmt.Errorf("collecting remote CPUs: %w", err)
	}
	hp.RemoteCPU = cpus

	memTotal, err := ProbeMemory(ctx, r)
	if err != nil {
		return hp, fmt.Errorf("collecting memory info: %w", err)
	}
	hp.TotalMemoryKb = memTotal

	return hp, nil
}
