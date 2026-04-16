package templates

import (
	"bytes"
	"encoding/json"
	"fmt"
	"text/template"

	"github.com/arm/topo/internal/target"
)

type PrintableTargetDescription struct {
	target.HardwareProfile
}

const describeTemplate = `Host Processors
---------------
{{- range .HostProcessor }}
  Model: {{ .Model }}
  Cores: {{ .Cores }}
  Features: {{ join .Features ", " }}
{{- end }}
{{ if .RemoteCPU }}
Remote Processors
-----------------
{{- range .RemoteCPU }}
  {{ .Name }}
{{- end }}
{{ end }}
Total Memory: {{ .TotalMemoryKb }} KB
`

func (d PrintableTargetDescription) AsPlain(isTTY bool) (string, error) {
	tmpl, err := template.
		New("describe").
		Funcs(getFuncMap(isTTY)).
		Parse(describeTemplate)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, d.HardwareProfile); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (d PrintableTargetDescription) AsJSON() (string, error) {
	b, err := json.MarshalIndent(d.HardwareProfile, "", "  ")
	if err != nil {
		return "", fmt.Errorf("encode target description as json: %w", err)
	}
	return string(b), nil
}
