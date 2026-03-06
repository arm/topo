package target

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var armCpuFeatures = map[string]string{
	"asimd": "NEON",
	"sve":   "SVE",
	"sve2":  "SVE2",
	"sme":   "SME",
	"sme2":  "SME2",
}

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

func (c *Connection) ProbeHardware() (HardwareProfile, error) {
	var hp HardwareProfile

	cpuProfile, err := c.collectCPUInfo()
	if err != nil {
		return hp, fmt.Errorf("collecting CPU info: %w", err)
	}
	hp.HostProcessor = cpuProfile

	cpus, err := c.ProbeRemoteproc()
	if err != nil {
		return hp, fmt.Errorf("collecting remote CPUs: %w", err)
	}
	hp.RemoteCPU = cpus

	memTotal, err := c.collectMemInfo()
	if err != nil {
		return hp, fmt.Errorf("collecting memory info: %w", err)
	}
	hp.TotalMemoryKb = memTotal

	return hp, nil
}

func (c *Connection) ProbeRemoteproc() ([]RemoteprocCPU, error) {
	var remoteProcs []RemoteprocCPU
	out, err := c.Run("ls /sys/class/remoteproc")
	if err != nil || out == "" {
		return remoteProcs, nil
	}

	out, err = c.Run("cat /sys/class/remoteproc/*/name")
	if err != nil {
		return remoteProcs, err
	}

	remoteCPU := strings.FieldsSeq(out)
	for cpu := range remoteCPU {
		remoteProcs = append(remoteProcs, RemoteprocCPU{Name: cpu})
	}
	return remoteProcs, nil
}

func (c *Connection) collectCPUInfo() ([]HostProcessor, error) {
	if err := c.BinaryExists("lscpu"); err != nil {
		return nil, err
	}

	out, err := c.Run("lscpu --json")
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

func (c *Connection) collectMemInfo() (int64, error) {
	key := "MemTotal"
	path := "/proc/meminfo"

	out, err := c.Run(fmt.Sprintf("cat %s", path))
	if err != nil {
		return 0, err
	}

	value, err := FindKeyValueInString(key, out)
	if err != nil {
		return 0, fmt.Errorf("in checking %s", path)
	}
	return value, nil
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
