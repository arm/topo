package views

import (
	"bytes"
	"encoding/json"
	"fmt"
	"text/tabwriter"
	"text/template"

	"github.com/arm/topo/cli/internal/deploy"
)

type ContainerList struct {
	Containers []deploy.Container `json:"containers"`
}

const containerListTemplate = `Container ID	Names	Image	Status	Processing Domain	Address
{{- range .}}
{{.Id}}	{{.Names}}	{{.Image}}	{{.Status}}	{{.ProcessingDomain}}	{{.Address}}
{{- end }}`

func (r ContainerList) AsPlain(isTTY bool) (string, error) {
	tmpl, err := template.
		New("ps").
		Parse(containerListTemplate)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	const columnPadding = 3
	w := tabwriter.NewWriter(&buf, 0, 0, columnPadding, ' ', 0)
	if err := tmpl.Execute(w, r.Containers); err != nil {
		return "", err
	}
	err = w.Flush()
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

func (r ContainerList) AsJSON() (string, error) {
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", fmt.Errorf("encode report as json: %w", err)
	}
	return string(b), nil
}
