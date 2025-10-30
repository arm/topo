package core

import (
	"bytes"
	"html/template"
	"os/exec"
	"slices"
	"strings"
)

const healthCheckTemplate = `Host
----
  SSH: {{.HostIcon}}
  {{- if .ShowTarget}}
Target
------
  Connected: {{.TargetIcon}}
  {{- if .Connected}}
  Features (Linux Host): {{.Features}}
  {{- end}}
  {{- end}}
`

var searchFlags = map[string]string{
	"asimd": "NEON",
	"sve":   "SVE",
	"sve2":  "SVE2",
	"sme":   "SME",
	"sme2":  "SME2",
}

func MakeHostPath() []string {
	res := []string{}
	searchBinaryFor := []string{"ssh", "docker", "podman"}

	for _, bin := range searchBinaryFor {
		if binaryExists(bin) {
			res = append(res, bin)
		}
	}
	return res
}

func binaryExists(bin string) bool {
	_, err := exec.LookPath(bin)
	return err == nil
}

func extractArmFeatures(target Target) []string {
	res := make([]string, 0)

	for _, field := range target.features {
		if name, ok := searchFlags[field]; ok {
			res = append(res, name)
		}
	}
	return res
}

func HealthCheckStringBuilder(hostPath []string, target Target) string {
	boolIcon := func(b bool) string {
		if b {
			return "✅"
		}
		return "❌"
	}

	data := struct {
		HostIcon   string
		ShowTarget bool
		TargetIcon string
		Connected  bool
		Features   string
	}{
		HostIcon:   boolIcon(slices.Contains(hostPath, "ssh")),
		ShowTarget: slices.Contains(hostPath, "ssh"),
		TargetIcon: boolIcon(target.connectionError == nil),
		Connected:  target.connectionError == nil,
		Features:   strings.Join(extractArmFeatures(target), ", "),
	}

	var buf bytes.Buffer
	tmpl := template.Must(template.New("health").Parse(healthCheckTemplate))
	if err := tmpl.Execute(&buf, data); err != nil {
		return "<template execution error: " + err.Error() + ">"
	}
	return buf.String()
}

func CheckHealth(sshTarget string) error {
	hostPath := MakeHostPath()
	target := MakeTarget(sshTarget, ExecSSH)
	LogPrintf(HealthCheckStringBuilder(hostPath, target))
	return nil
}
