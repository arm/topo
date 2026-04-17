package probe

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"github.com/arm/topo/internal/runner"
)

type HostProcessor struct {
	Model    string   `yaml:"model" json:"model"`
	Cores    int      `yaml:"cores" json:"cores"`
	Features []string `yaml:"features" json:"features"`
}

var armCpuFeatures = map[string]string{
	"asimd": "NEON",
	"sve":   "SVE",
	"sve2":  "SVE2",
	"sme":   "SME",
	"sme2":  "SME2",
}

func (proc *HostProcessor) ExtractArmFeatures() []string {
	if len(proc.Features) == 0 {
		return nil
	}

	var res []string
	for _, field := range proc.Features {
		if name, ok := armCpuFeatures[field]; ok {
			res = append(res, name)
		}
	}
	return res
}

func CPU(ctx context.Context, r runner.Runner) ([]HostProcessor, error) {
	if err := r.BinaryExists(ctx, "lscpu"); err != nil {
		return nil, err
	}

	out, err := r.Run(ctx, "lscpu --json")
	if err != nil {
		return nil, err
	}

	var output lscpuJSON
	if err := json.Unmarshal([]byte(out), &output); err != nil {
		return nil, err
	}

	var processors []HostProcessor
	for name, fields := range groupByModelName(output.Lscpu) {
		hp, err := newHostProcessor(name, fields)
		if err != nil {
			return nil, err
		}
		processors = append(processors, hp)
	}
	return processors, nil
}

type lscpuOutputField struct {
	Field string `json:"field"`
	Data  string `json:"data"`
}

type lscpuJSON struct {
	Lscpu []lscpuOutputField `json:"lscpu"`
}

type fieldsByModelName map[string][]lscpuOutputField

func groupByModelName(fields []lscpuOutputField) fieldsByModelName {
	grouped := make(fieldsByModelName)
	currentName := ""
	for _, f := range fields {
		if f.Field == "Model name:" {
			currentName = f.Data
			continue
		}
		if currentName != "" {
			grouped[currentName] = append(grouped[currentName], f)
		}
	}
	return grouped
}

func newHostProcessor(name string, fields []lscpuOutputField) (HostProcessor, error) {
	coresPerUnit := 1
	units := 1
	foundUnits := false
	var features []string

	for _, f := range fields {
		switch f.Field {
		case "Core(s) per socket:", "Core(s) per cluster:":
			v, err := strconv.Atoi(f.Data)
			if err != nil {
				return HostProcessor{}, err
			}
			coresPerUnit = v
		case "Socket(s):", "Cluster(s):":
			v, err := strconv.Atoi(f.Data)
			if err != nil {
				// Some platforms report "-" for sockets when using clusters
				continue
			}
			units = v
			foundUnits = true
		case "Flags:":
			features = strings.Split(f.Data, " ")
		}
	}

	if !foundUnits {
		return HostProcessor{}, errors.New("could not determine CPU units")
	}

	return HostProcessor{
		Model:    name,
		Cores:    coresPerUnit * units,
		Features: features,
	}, nil
}
