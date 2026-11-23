package main

import (
	"bufio"
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"

	sdk "github.com/arm/topo/pkg/plugin"
	"github.com/arm/topo/pkg/plugin/observations"
)

func main() {
	sdk.ServeObservations("example.memory", provider{})
}

type provider struct{}

func (provider) ObserveTarget(ctx context.Context, runner observations.TargetRunner) (observations.Tree, error) {
	output, _, err := runner.Run(ctx, "cat /proc/meminfo")
	if err != nil {
		if ctx.Err() != nil {
			return observations.Tree{}, ctx.Err()
		}
		// Unavailable memory information is not evidence of zero memory.
		return observations.Tree{}, nil
	}
	bytes, err := totalMemoryBytes(output)
	if err != nil {
		return observations.Tree{}, fmt.Errorf("failed to read /proc/meminfo: %w", err)
	}
	return observations.Tree{
		// Kernel MemTotal in bytes, not a workload's cgroup memory limit.
		"memory": observations.Tree{"total_bytes": bytes},
	}, nil
}

func totalMemoryBytes(text string) (int64, error) {
	scanner := bufio.NewScanner(strings.NewReader(text))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) == 0 || fields[0] != "MemTotal:" {
			continue
		}
		if len(fields) != 3 || fields[2] != "kB" {
			return 0, fmt.Errorf("MemTotal must use Linux kB units (1024 bytes)")
		}
		kibibytes, err := strconv.ParseInt(fields[1], 10, 64)
		if err != nil {
			return 0, err
		}
		if kibibytes < 0 || kibibytes > math.MaxInt64/1024 {
			return 0, fmt.Errorf("MemTotal is negative or overflows bytes: %q", fields[1])
		}
		return kibibytes * 1024, nil
	}
	if err := scanner.Err(); err != nil {
		return 0, err
	}
	return 0, fmt.Errorf("field MemTotal not found")
}
