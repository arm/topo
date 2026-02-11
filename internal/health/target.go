package health

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/arm-debug/topo-cli/internal/ssh"
)

type execSSH func(target ssh.Host, command string) (string, error)

type HostProcessor struct {
	ModelName string   `yaml:"model"`
	Cores     int      `yaml:"cores"`
	Features  []string `yaml:"features"`
}

type RemoteprocCPU struct {
	Name string `yaml:"name"`
}

type HardwareProfile struct {
	HostProcessor []HostProcessor `yaml:"host"`
	RemoteCPU     []RemoteprocCPU `yaml:"remoteprocs"`
}

type LscpuOutputField struct {
	Field    string             `json:"field"`
	Data     string             `json:"data"`
	Children []LscpuOutputField `json:"children,omitempty"`
}

type LscpuOutput struct {
	Lscpu []LscpuOutputField `json:"lscpu"`
}

func (hw HardwareProfile) Capabilities() map[HardwareCapability]struct{} {
	capabilities := make(map[HardwareCapability]struct{})
	if len(hw.RemoteCPU) > 0 {
		capabilities[Remoteproc] = struct{}{}
	}
	return capabilities
}

type Status struct {
	SSHTarget       ssh.Host
	ConnectionError error
	Dependencies    []DependencyStatus
	Hardware        HardwareProfile
}

type Connection struct {
	sshTarget ssh.Host
	exec      execSSH
}

func NewConnection(sshTarget string, exec execSSH) Connection {
	return Connection{
		sshTarget: ssh.Host(sshTarget),
		exec:      exec,
	}
}

func (c *Connection) Run(command string) (string, error) {
	return c.exec(c.sshTarget, command)
}

func (c *Connection) BinaryExists(bin string) (bool, error) {
	if err := ssh.ValidateBinaryName(bin); err != nil {
		return false, err
	}
	_, err := c.exec(c.sshTarget, ssh.ShellCommand(fmt.Sprintf("command -v %s", bin)))
	return err == nil, nil
}

func (c *Connection) Probe() Status {
	var targetStatus Status
	targetStatus.SSHTarget = c.sshTarget

	if err := c.ProbeConnection(); err != nil {
		targetStatus.ConnectionError = err
		return targetStatus
	}

	targetStatus.Hardware, _ = c.ProbeHardware()
	targetStatus.Dependencies = c.CheckDependencies(targetStatus.Hardware.Capabilities())

	return targetStatus
}

func (c *Connection) ProbeConnection() error {
	_, err := c.Run("true")
	return err
}

func (c *Connection) CheckDependencies(hardware map[HardwareCapability]struct{}) []DependencyStatus {
	deps := FilterByHardware(TargetRequiredDependencies, hardware)
	return CheckInstalled(deps, c.BinaryExists)
}

func (c *Connection) ProbeHardware() (HardwareProfile, error) {
	var hp HardwareProfile

	cpuProfile, err := c.collectCPUInfo()
	if err != nil {
		return hp, fmt.Errorf("collecting CPU info: %w", err)
	}
	hp.HostProcessor = cpuProfile

	cpus, err := c.collectRemoteCPU()
	if err != nil {
		return hp, fmt.Errorf("collecting remote CPUs: %w", err)
	}
	hp.RemoteCPU = cpus

	return hp, nil
}

func (c *Connection) collectCPUInfo() ([]HostProcessor, error) {
	ok, err := c.BinaryExists("lscpu")
	if err != nil {
		return nil, fmt.Errorf("checking for lscpu: %w", err)
	}
	if !ok {
		return nil, errors.New("lscpu not found")
	}

	out, err := c.Run("lscpu --json")
	if err != nil {
		return nil, err
	}

	var lscpu LscpuOutput
	err = json.Unmarshal([]byte(out), &lscpu)
	if err != nil {
		return nil, err
	}

	return CreateCPUProfile(lscpu.Lscpu)
}

func newHostProcessor(name string, fields []LscpuOutputField) (HostProcessor, error) {
	coresPerUnit := 1
	units := 1
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
		case "Flags:":
			features = strings.Split(f.Data, " ")
		}
	}

	return HostProcessor{
		ModelName: name,
		Cores:     coresPerUnit * units,
		Features:  features,
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

func (c *Connection) collectRemoteCPU() ([]RemoteprocCPU, error) {
	out, err := c.Run("ls /sys/class/remoteproc")
	if err != nil || out == "" {
		return nil, nil
	}

	out, err = c.Run("cat /sys/class/remoteproc/*/name")
	if err != nil {
		return nil, err
	}

	remoteCPU := strings.Fields(out)
	var remoteProcs []RemoteprocCPU
	for _, cpu := range remoteCPU {
		remoteProcs = append(remoteProcs, RemoteprocCPU{Name: cpu})
	}
	return remoteProcs, nil
}
