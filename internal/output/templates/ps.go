package templates

import (
	"bytes"
	"encoding/json"
	"fmt"
	"text/template"
)

type ContainerStatus struct {
	Image  string `json:"Image"`
	Status string `json:"Status"`
	Ports  string `json:"Ports"`
}

type PrintablePSReport struct {
	Containers []ContainerStatus `json:"containers"`
	Target     string            `json:"targetHost"`
}

const PSTemplate = `
{{ range .}}
{{.Image}}
{{ end }}
`

func (r PrintablePSReport) AsPlain(isTTY bool) (string, error) {
	tmpl, err := template.
		New("ps").
		Parse(PSTemplate)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, r.Containers); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func (r PrintablePSReport) AsJSON() (string, error) {
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", fmt.Errorf("encode report as json: %w", err)
	}
	return string(b), nil
}
