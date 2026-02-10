package health

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/arm-debug/topo-cli/internal/ssh"
)

type execSSH func(target ssh.Host, command string) (string, error)

type HostCPUProfile struct {
	ModelName string   `yaml:"model"`
	Cores     int      `yaml:"cores"`
	Features  []string `yaml:"features"`
}

type RemoteProcCPU struct {
	Name string `yaml:"name"`
}

type HardwareProfile struct {
	HostCPU   HostCPUProfile  `yaml:"host"`
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

	if cpuProfile, err := c.collectCPUInfo(); err == nil {
		hp.HostCPU = cpuProfile
	}
	if cpus, err := c.collectRemoteCPU(); err == nil {
		hp.RemoteCPU = cpus
	}

	return hp, nil
}

func (c *Connection) collectCPUInfo() (HostCPUProfile, error) {
	if ok, err := c.BinaryExists("lscpu"); !ok || err != nil {
		return HostCPUProfile{}, fmt.Errorf("lscpu not found or not accessible")
	}

	out, err := c.Run("lscpu --json")
	if err != nil {
		return HostCPUProfile{}, err
	}

	var lscpu LscpuOutput
	err = json.Unmarshal([]byte(out), &lscpu)
	if err != nil {
		return HostCPUProfile{}, err
	}

	return createCPUProfile(flattenLscpuFields(lscpu.Lscpu)), nil
}

func flattenLscpuFields(fields []LscpuOutputField) []LscpuOutputField {
	var res []LscpuOutputField
	for _, field := range fields {
		res = append(res, field)
		if len(field.Children) > 0 {
			res = append(res, flattenLscpuFields(field.Children)...)
		}
	}
	return res
}

func createCPUProfile(fields []LscpuOutputField) HostCPUProfile {
	var cpuProfile HostCPUProfile

	for _, field := range fields {
		switch field.Field {
		case "Model name:":
			cpuProfile.ModelName = field.Data
		case "CPU(s):":
			_, _ = fmt.Sscanf(field.Data, "%d", &cpuProfile.Cores)
		case "Flags:":
			cpuProfile.Features = strings.Fields(field.Data)
		}
	}

	return cpuProfile
}

func (c *Connection) collectRemoteCPU() ([]RemoteProcCPU, error) {
	out, err := c.Run("ls /sys/class/remoteproc")
	if err != nil {
		return nil, err
	}

	if out == "" {
		return nil, fmt.Errorf("target supports remoteproc, but no processors found")
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
