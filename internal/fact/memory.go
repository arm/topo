package fact

import (
	"bufio"
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/arm/topo/internal/runner"
)

const (
	GroupSystemMemory = "system-memory"
	FactMemoryTotal   = "memory.total"
)

type MemoryProvider struct{}

func (MemoryProvider) Facts(ctx context.Context, r runner.CommandRunner) (Report, error) {
	path := "/proc/meminfo"
	out, _, err := r.Run(ctx, fmt.Sprintf("cat %s", path))
	if err != nil {
		return Report{}, err
	}

	value, err := findKeyValue("MemTotal", out)
	if err != nil {
		return Report{}, fmt.Errorf("failed to read %s: %w", path, err)
	}
	return Report{
		Groups: []Group{{
			ID: GroupSystemMemory,
			Facts: []Fact{{
				Key:         FactMemoryTotal,
				Value:       Value{Type: "integer", Integer: value},
				Label:       "Total memory",
				Description: "Kernel-reported MemTotal in KiB",
			}},
		}},
	}, nil
}

func findKeyValue(key string, text string) (int64, error) {
	scanner := bufio.NewScanner(strings.NewReader(text))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 2 && fields[0] == key+":" {
			return strconv.ParseInt(fields[1], 10, 64)
		}
	}
	return 0, fmt.Errorf("field %s not found", key)
}
