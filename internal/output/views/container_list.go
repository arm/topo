package views

import (
	"bytes"
	"encoding/json"
	"fmt"
	"text/tabwriter"
	"text/template"
)

type Container struct {
	ID               string `json:"id"`
	Names            string `json:"names"`
	Image            string `json:"image"`
	State            string `json:"state"`
	Status           string `json:"status"`
	ProcessingDomain string `json:"processingDomain"`
	Address          string `json:"address"`
}

type ContainerList struct {
	Containers []Container `json:"containers"`
}

const containerListTemplate = `Container ID	Names	Image	Status	Processing Domain	Address
{{- range .}}
{{.ID}}	{{.Names}}	{{.Image}}	{{.Status}}	{{.ProcessingDomain}}	{{.Address}}
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
