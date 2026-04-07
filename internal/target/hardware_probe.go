package target

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/arm/topo/internal/command"
	"github.com/arm/topo/internal/runner"
)

type HostProcessor struct {
	Model    string   `yaml:"model"`
	Cores    int      `yaml:"cores"`
	Features []string `yaml:"features"`
}

type RemoteprocCPU struct {
	Name string `yaml:"name"`
}

type HardwareProfile struct {
	HostProcessor []HostProcessor `yaml:"host"`
	RemoteCPU     []RemoteprocCPU `yaml:"remoteprocs"`
	TotalMemoryKb int64           `yaml:"totalmemory_kb"`
}

type HardwareProbe struct {
	runner runner.Runner
}

func NewHardwareProbe(r runner.Runner) HardwareProbe {
	return HardwareProbe{runner: r}
}

func (p *HardwareProbe) Probe(ctx context.Context) (HardwareProfile, error) {
	var hp HardwareProfile

	cpuProfile, err := p.collectCPUInfo(ctx)
	if err != nil {
		return hp, fmt.Errorf("collecting CPU info: %w", err)
	}
	hp.HostProcessor = cpuProfile

	cpus, err := p.ProbeRemoteproc(ctx)
	if err != nil {
		return hp, fmt.Errorf("collecting remote CPUs: %w", err)
	}
	hp.RemoteCPU = cpus

	memTotal, err := p.collectMemInfo(ctx)
	if err != nil {
		return hp, fmt.Errorf("collecting memory info: %w", err)
	}
	hp.TotalMemoryKb = memTotal

	return hp, nil
}

func (p *HardwareProbe) ProbeRemoteproc(ctx context.Context) ([]RemoteprocCPU, error) {
	var remoteProcs []RemoteprocCPU
	out, err := p.runner.Run(ctx, command.WrapInLoginShell("ls /sys/class/remoteproc"))
	if err != nil || out == "" {
		return remoteProcs, nil
	}

	out, err = p.runner.Run(ctx, command.WrapInLoginShell("cat /sys/class/remoteproc/*/name"))
	if err != nil {
		return remoteProcs, err
	}

	remoteCPU := strings.FieldsSeq(out)
	for cpu := range remoteCPU {
		remoteProcs = append(remoteProcs, RemoteprocCPU{Name: cpu})
	}
	return remoteProcs, nil
}

func (p *HardwareProbe) collectCPUInfo(ctx context.Context) ([]HostProcessor, error) {
	if err := p.runner.BinaryExists("lscpu"); err != nil {
		return nil, err
	}

	out, err := p.runner.Run(ctx, command.WrapInLoginShell("lscpu --json"))
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

func (p *HardwareProbe) collectMemInfo(ctx context.Context) (int64, error) {
	key := "MemTotal"
	path := "/proc/meminfo"

	out, err := p.runner.Run(ctx, command.WrapInLoginShell(fmt.Sprintf("cat %s", path)))
	if err != nil {
		return 0, err
	}

	value, err := FindKeyValueInString(key, out)
	if err != nil {
		return 0, fmt.Errorf("in checking %s", path)
	}
	return value, nil
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

func FindKeyValueInString(key string, text string) (int64, error) {
	scanner := bufio.NewScanner(strings.NewReader(text))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 2 && fields[0] == key+":" {
			return strconv.ParseInt(fields[1], 10, 64)
		}
	}
	return 0, fmt.Errorf("field %s not found", key)
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
