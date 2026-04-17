package target

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

func ProbeCPU(ctx context.Context, r runner.Runner) ([]HostProcessor, error) {
	if err := r.BinaryExists(ctx, "lscpu"); err != nil {
		return nil, err
	}

	out, err := r.Run(ctx, "lscpu --json")
	if err != nil {
		return nil, err
	}

	var lscpuOutput lscpuOutput
	err = json.Unmarshal([]byte(out), &lscpuOutput)
	if err != nil {
		return nil, err
	}

	return CreateCPUProfile(lscpuOutput.Lscpu)
}

var armCpuFeatures = map[string]string{
	"asimd": "NEON",
	"sve":   "SVE",
	"sve2":  "SVE2",
	"sme":   "SME",
	"sme2":  "SME2",
}

type LscpuOutputField struct {
	Field    string             `json:"field"`
	Data     string             `json:"data"`
	Children []LscpuOutputField `json:"children,omitempty"`
}

type lscpuOutput struct {
	Lscpu []LscpuOutputField `json:"lscpu"`
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

func newHostProcessor(name string, fields []LscpuOutputField) (HostProcessor, error) {
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

func CreateCPUProfile(fields []LscpuOutputField) ([]HostProcessor, error) {
	type coreType struct {
		name   string
		fields []LscpuOutputField
	}
	var coreTypes []coreType

	for _, f := range fields {
		if f.Field == "Model name:" {
			coreTypes = append(coreTypes, coreType{name: f.Data})
			continue
		}
		if len(coreTypes) > 0 {
			coreTypes[len(coreTypes)-1].fields = append(coreTypes[len(coreTypes)-1].fields, f)
		}
	}

	var profiles []HostProcessor
	for _, g := range coreTypes {
		hp, err := newHostProcessor(g.name, g.fields)
		if err != nil {
			return nil, err
		}
		profiles = append(profiles, hp)
	}
	return profiles, nil
}
