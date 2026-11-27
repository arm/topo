package fact

import (
	"context"
	"fmt"
	"maps"
	"path"
	"slices"
	"strings"

	"github.com/arm/topo/internal/runner"
)

const remoteprocNameField = "name"

type RemoteprocProvider struct{}

func (RemoteprocProvider) Facts(ctx context.Context, r runner.CommandRunner) (Report, error) {
	out, stderr, err := r.Run(ctx, `sh -c 'LC_ALL=C grep -H ^ /sys/class/remoteproc/*/name'`)
	if err != nil {
		if strings.Contains(stderr, "No such file or directory") {
			return nil, nil
		}
		return nil, err
	}

	report := make(Report)
	for line := range strings.SplitSeq(out, "\n") {
		if line == "" {
			continue
		}
		deviceID, name, err := parseRemoteprocLine(line)
		if err != nil {
			return nil, err
		}
		key := deviceID + "." + remoteprocNameField
		if _, exists := report[key]; exists {
			return nil, fmt.Errorf("duplicate remoteproc fact key: %q", key)
		}
		report[key] = Fact{
			Value:       Value{Type: "string", Text: name},
			Label:       "Remote processor name",
			Description: "Kernel-reported remote processor name",
		}
	}
	return report, nil
}

func RemoteprocNames(report Report) []string {
	var names []string
	for _, key := range slices.Sorted(maps.Keys(report)) {
		deviceID, field, _ := strings.Cut(key, ".")
		if deviceID != "" && field == remoteprocNameField {
			names = append(names, report[key].Value.String())
		}
	}
	return names
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
