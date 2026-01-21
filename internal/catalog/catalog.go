package catalog

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"text/template"
)

//go:embed data/templates.json
var TemplatesJSON []byte

const (
	reset = "\033[0m"
	red   = "\033[31m"
	green = "\033[32m"
)

type Repo struct {
	Id          string   `json:"id"`
	Description string   `json:"description"`
	Features    []string `json:"features"`
	Url         string   `json:"url"`
	Ref         string   `json:"ref"`
}

func GetTemplateRepo(id string) (*Repo, error) {
	return GetRepo(id, TemplatesJSON)
}

func PrintTemplateRepos(w io.Writer, templatesJSON []byte) error {
	repos, err := ListRepos(templatesJSON)
	if err != nil {
		return err
	}
	for _, repo := range repos {
		if err := printRepo(w, repo); err != nil {
			return err
		}
	}
	return err
}

func ListRepos(b []byte) ([]Repo, error) {
	var templates []Repo
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&templates); err != nil {
		return nil, fmt.Errorf("failed to unmarshal templates: %w", err)
	}
	return templates, nil
}

func GetRepo(id string, b []byte) (*Repo, error) {
	repos, err := ListRepos(b)
	if err != nil {
		return nil, err
	}
	for i := range repos {
		if repos[i].Id == id {
			return &repos[i], nil
		}
	}
	return nil, fmt.Errorf("repo with id %q not found", id)
}

const repoTemplate = `
{{- define "featuresRow" -}}
{{- if .Features }}
  Features: {{ join .Features ", " }}
{{- end }}
{{- end }}

{{- define "descriptionRow"}}
{{- if .Description }}
{{ wrap .Description }}
{{ end }}
{{- end }}

{{- green .Id }} | {{ red .Url }} | {{ red .Ref }}
{{- template "featuresRow" . }}
{{- template "descriptionRow" . }}
`

var baseRepoTemplate = template.Must(
	template.New("repo").
		Funcs(template.FuncMap{
			"join":  strings.Join,
			"wrap":  func(s string) string { return WrapText(s, 80, 2) },
			"green": func(s string) string { return s },
			"red":   func(s string) string { return s },
		}).
		Parse(repoTemplate),
)

func isTTY(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}

	info, err := f.Stat()
	if err != nil {
		return false
	}

	return (info.Mode() & os.ModeCharDevice) != 0
}

func colour(w io.Writer, col, str string) string {
	if !isTTY(w) {
		return str
	}
	return col + str + reset
}

func printRepo(w io.Writer, r Repo) error {
	tmpl, err := baseRepoTemplate.Clone()
	if err != nil {
		return err
	}

	tmpl = tmpl.Funcs(template.FuncMap{
		"green": func(s string) string { return colour(w, green, s) },
		"red":   func(s string) string { return colour(w, red, s) },
	})

	return tmpl.Execute(w, r)
}

func WrapText(s string, maxWidth, indentSpaces int) string {
	if maxWidth <= 0 {
		return s
	}
	if indentSpaces < 0 {
		indentSpaces = 0
	}

	var out []string
	prefix := strings.Repeat(" ", indentSpaces)
	for para := range strings.SplitSeq(s, "\n\n") {
		for rawLine := range strings.SplitSeq(para, "\n") {
			line := prefix

			for word := range strings.FieldsSeq(rawLine) {
				space := 1
				if line == prefix {
					space = 0
				}

				if len(line)+space+len(word) > maxWidth {
					out = append(out, line)
					line = prefix + word
				} else {
					if line != prefix {
						line += " "
					}
					line += word
				}
			}

			if line != prefix {
				out = append(out, line)
			}
		}

		out = append(out, "")
	}
	if len(out) > 0 {
		out = out[:len(out)-1]
	}
	return strings.Join(out, "\n")
}
