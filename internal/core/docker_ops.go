package core

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Docker ps / inspect derived types
type DockerPsItem struct {
	Command      string `json:"Command"`
	CreatedAt    string `json:"CreatedAt"`
	ID           string `json:"ID"`
	Image        string `json:"Image"`
	Labels       string `json:"Labels"`
	LocalVolumes string `json:"LocalVolumes"`
	Mounts       string `json:"Mounts"`
	Names        string `json:"Names"`
	Networks     string `json:"Networks"`
	Ports        string `json:"Ports"`
	RunningFor   string `json:"RunningFor"`
	Size         string `json:"Size"`
	State        string `json:"State"`
	Status       string `json:"Status"`
}

type DockerPsItemWithRuntime struct {
	DockerPsItem
	Runtime string `json:"Runtime"`
	Ports   []int  `json:"HostPorts"`
}

// ReadContainersInfo returns enriched ps output.
func ReadContainersInfo(sshTarget string) ([]DockerPsItemWithRuntime, error) {
	conn := []string{"-H", fmt.Sprintf("ssh://%s", sshTarget)}
	cmd := ExecCommand("docker", append(conn, "ps", "-a", "--format", "{{json .}}")...)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	lines := FilterNonEmpty(strings.Split(strings.TrimSpace(string(out)), "\n"))
	if len(lines) == 0 {
		return []DockerPsItemWithRuntime{}, nil
	}
	items := make([]DockerPsItem, 0, len(lines))
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if !strings.HasPrefix(l, "{") {
			continue
		}
		var itm DockerPsItem
		_ = json.Unmarshal([]byte(l), &itm)
		items = append(items, itm)
	}
	ids := make([]string, len(items))
	for i, itm := range items {
		ids[i] = itm.ID
	}
	if len(ids) == 0 {
		return []DockerPsItemWithRuntime{}, nil
	}
	inspectArgs := append(append(conn, "inspect"), ids...)
	inspectArgs = append(inspectArgs, "--format", `{{json .NetworkSettings.Ports}};{{.HostConfig.Runtime}}`)
	inspectCmd := ExecCommand("docker", inspectArgs...)
	inspectOut, err := inspectCmd.Output()
	if err != nil {
		return nil, err
	}
	inspectLines := FilterNonEmpty(strings.Split(strings.TrimSpace(string(inspectOut)), "\n"))
	if len(inspectLines) != len(items) {
		return nil, fmt.Errorf("mismatch between ps items and inspect lines")
	}
	result := make([]DockerPsItemWithRuntime, len(items))
	for i, itm := range items {
		parts := strings.SplitN(inspectLines[i], ";", 2)
		var portsJSON, runtimeStr string
		if len(parts) >= 1 {
			portsJSON = parts[0]
		}
		if len(parts) == 2 {
			runtimeStr = parts[1]
		}
		hostPorts, _ := ParsePorts(portsJSON)
		if hostPorts == nil {
			hostPorts = []int{}
		}
		result[i] = DockerPsItemWithRuntime{DockerPsItem: itm, Runtime: runtimeStr, Ports: hostPorts}
	}
	return result, nil
}

// ParsePorts extracts host ports for container ports 80/443 from docker inspect port JSON.
func ParsePorts(portsJSON string) ([]int, error) {
	var portMap map[string][]struct {
		HostPort string `json:"HostPort"`
	}
	if err := json.Unmarshal([]byte(portsJSON), &portMap); err != nil {
		return nil, err
	}
	portSet := map[int]struct{}{}
	for key, mappings := range portMap {
		portStr := strings.Split(key, "/")[0]
		p, _ := strconv.Atoi(portStr)
		if (p == 80 || p == 443) && len(mappings) > 0 {
			for _, m := range mappings {
				if m.HostPort != "" {
					if hp, err := strconv.Atoi(m.HostPort); err == nil {
						portSet[hp] = struct{}{}
					}
				}
			}
		}
	}
	out := make([]int, 0, len(portSet))
	for hp := range portSet {
		out = append(out, hp)
	}
	return out, nil
}

// FilterNonEmpty removes blank lines.
func FilterNonEmpty(ss []string) []string {
	ret := make([]string, 0, len(ss))
	for _, s := range ss {
		if t := strings.TrimSpace(s); t != "" {
			ret = append(ret, t)
		}
	}
	return ret
}

func PrintContainersInfo(w io.Writer, sshTarget string) error {
	items, err := ReadContainersInfo(sshTarget)
	if err != nil {
		return fmt.Errorf("failed to read containers info: %w", err)
	}
	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal containers info: %w", err)
	}
	fmt.Fprintf(w, "%s\n", data)
	return nil
}
