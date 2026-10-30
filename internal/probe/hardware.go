package probe

import (
	"context"
	"fmt"

	"github.com/arm/topo/internal/fact"
	"github.com/arm/topo/internal/runner"
)

type HardwareProfile struct {
	HostProcessors   []HostProcessor   `yaml:"hostProcessors" json:"hostProcessors"`
	RemoteProcessors []RemoteProcessor `yaml:"remoteProcessors" json:"remoteProcessors,omitempty"`
	TotalMemoryKb    int64             `yaml:"totalMemoryKb" json:"totalMemoryKb"`
}

type RemoteProcessor struct {
	Name string `yaml:"name" json:"name"`
}

func Hardware(ctx context.Context, r runner.Runner) (HardwareProfile, error) {
	var hp HardwareProfile

	hostProcessors, err := HostProcessors(ctx, r)
	if err != nil {
		return hp, fmt.Errorf("collecting CPU info: %w", err)
	}
	hp.HostProcessors = hostProcessors

	remoteProcessors, err := remoteProcessors(ctx, r)
	if err != nil {
		return hp, fmt.Errorf("collecting remote processors: %w", err)
	}
	hp.RemoteProcessors = remoteProcessors

	memTotal, err := memoryTotal(ctx, r)
	if err != nil {
		return hp, fmt.Errorf("collecting memory info: %w", err)
	}
	hp.TotalMemoryKb = memTotal

	return hp, nil
}

func memoryTotal(ctx context.Context, r runner.Runner) (int64, error) {
	provider := fact.MemoryProvider{}
	report, err := provider.Facts(ctx, r)
	if err != nil {
		return 0, err
	}
	memory, ok := report.Find(fact.GroupSystemMemory, fact.FactMemoryTotal)
	if !ok || memory.Value.Type != "integer" {
		return 0, fmt.Errorf("memory provider did not return an integer %s fact in %s", fact.FactMemoryTotal, fact.GroupSystemMemory)
	}
	return memory.Value.Integer, nil
}

func remoteProcessors(ctx context.Context, r runner.Runner) ([]RemoteProcessor, error) {
	provider := fact.RemoteprocProvider{}
	report, err := provider.Facts(ctx, r)
	if err != nil {
		return nil, err
	}

	var remoteProcs []RemoteProcessor
	for _, group := range report.Groups {
		name, ok := group.Find(fact.FactRemoteprocName)
		if !ok {
			return nil, fmt.Errorf("remoteproc provider did not return a %s for %s", fact.FactRemoteprocName, group.ID)
		}
		remoteProcs = append(remoteProcs, RemoteProcessor{Name: name.Value.String()})
	}
	return remoteProcs, nil
}
