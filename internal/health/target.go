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

type RemoteProcCPU struct {
	Name string `yaml:"name"`
}

type HardwareProfile struct {
	HostProcessor []HostProcessor  `yaml:"host"`
	RemoteCPU []RemoteProcCPU `yaml:"remoteprocs"`
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
	if err != nil{
		return nil, fmt.Errorf("checking for lscpu: %w", err)
	}
	if !ok {
		return nil, errors.New("lscpu not found.")
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

func newHostProcessor(fields LscpuOutputField) (HostProcessor, error) {
	var coresPerUnit, units int
	var features []string

	for _, f := range fields.Children {
		if f.Field == "Model name:" {
			break
		}
		switch f.Field {
		case "Core(s) per socket:":
			v, err := strconv.Atoi(f.Data)
			if err != nil {
				return HostProcessor{}, err
			}
			coresPerUnit = v
		case "Socket(s):":
			v, err := strconv.Atoi(f.Data)
			if err != nil {
				// Some platforms report "-" for sockets when using clusters
				continue
			}
			units = v
		case "Core(s) per cluster:":
			v, err := strconv.Atoi(f.Data)
			if err != nil {
				return HostProcessor{}, err
			}
			coresPerUnit = v
		case "Cluster(s):":
			v, err := strconv.Atoi(f.Data)
			if err != nil {
				return HostProcessor{}, err
			}
			units = v
		case "Flags:":
			features = strings.Split(f.Data, " ")
		}
	}

	return HostProcessor{
		ModelName: fields.Data,
		Cores:     coresPerUnit * units,
		Features:  features,
	}, nil
}

func CreateCPUProfile(fields []LscpuOutputField) ([]HostProcessor, error) {
	var profiles []HostProcessor
	for i, f := range fields {
		if f.Field != "Model name:" {
			continue
		}
		proc := LscpuOutputField{
			Field:    f.Field,
			Data:     f.Data,
			Children: fields[i+1:],
		}
		hp, err := newHostProcessor(proc)
		if err != nil {
			return nil, err
		}
		profiles = append(profiles, hp)
	}
	return profiles, nil
}

func (c *Connection) collectRemoteCPU() ([]RemoteProcCPU, error) {
	out, err := c.Run("ls /sys/class/remoteproc")
	if err != nil || out == "" {
		return nil, nil
	}

	out, err = c.Run("cat /sys/class/remoteproc/*/name")
	if err != nil {
		return nil, err
	}

	remoteCPU := strings.Fields(out)
	var remoteProcs []RemoteProcCPU
	for _, cpu := range remoteCPU {
		remoteProcs = append(remoteProcs, RemoteProcCPU{Name: cpu})
	}
	return remoteProcs, nil
}
