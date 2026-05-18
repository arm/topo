package templates

import (
	"bytes"
	"encoding/json"
	"fmt"
	"text/tabwriter"
	"text/template"
)

type ContainerStatus struct {
	Image  string `json:"Image"`
	Status string `json:"Status"`
	Ports  string `json:"Ports"`
}

type PrintablePSReport struct {
	Containers []ContainerStatus `json:"containers"`
}

const PSTemplate = `{{if .}}Image	Status	Ports
{{- range .}}
{{.Image}}	{{.Status}}	{{.Ports}}
{{- end }}{{else}}No containers deployed from this project are running.{{end}}`

func (r PrintablePSReport) AsPlain(isTTY bool) (string, error) {
	tmpl, err := template.
		New("ps").
		Parse(PSTemplate)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	w := tabwriter.NewWriter(&buf, 0, 0, 3, ' ', 0)
	if err := tmpl.Execute(w, r.Containers); err != nil {
		return "", err
	}
	err = w.Flush()
	if err != nil {
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
