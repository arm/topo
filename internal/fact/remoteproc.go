package fact

import (
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/arm/topo/internal/runner"
)

const FactRemoteprocName = "remoteproc.name"

type RemoteprocProvider struct{}

func (RemoteprocProvider) Facts(ctx context.Context, r runner.CommandRunner) (Report, error) {
	out, stderr, err := r.Run(ctx, `sh -c 'LC_ALL=C grep -H ^ /sys/class/remoteproc/*/name'`)
	if err != nil {
		if strings.Contains(stderr, "No such file or directory") {
			return Report{}, nil
		}
		return Report{}, err
	}

	var report Report
	for line := range strings.SplitSeq(out, "\n") {
		if line == "" {
			continue
		}
		filename, name, err := parseRemoteprocLine(line)
		if err != nil {
			return Report{}, err
		}
		report.Groups = append(report.Groups, Group{
			ID: filename,
			Facts: []Fact{{
				Key:         FactRemoteprocName,
				Value:       Value{Type: "string", Text: name},
				Label:       "Remote processor name",
				Description: "Kernel-reported remote processor name",
			}},
		})
	}
	return report, nil
}

func parseRemoteprocLine(line string) (string, string, error) {
	filename, name, ok := strings.Cut(line, ":")
	if !ok {
		return "", "", fmt.Errorf("invalid remoteproc entry: %q", line)
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return "", "", fmt.Errorf("empty remoteproc name in line: %q", line)
	}
	return path.Base(path.Dir(filename)), name, nil
}
