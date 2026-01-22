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
	reset  = "\033[0m"
	yellow = "\033[33m"
	blue   = "\033[34m"
	cyan   = "\033[36m"
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
	repos, err := ParseRepos(templatesJSON)
	if err != nil {
		return err
	}
	for _, repo := range repos {
		if err := printRepo(w, repo); err != nil {
			return fmt.Errorf("failed to load template catalog: %w", err)
		}
	}
	return err
}

func ParseRepos(b []byte) ([]Repo, error) {
	var templates []Repo
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&templates); err != nil {
		return nil, fmt.Errorf("failed to unmarshal templates: %w", err)
	}
	return templates, nil
}

func GetRepo(id string, b []byte) (*Repo, error) {
	repos, err := ParseRepos(b)
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

{{- cyan .Id }} | {{ blue .Url }} | {{ yellow .Ref }}
{{- template "featuresRow" . }}
{{- template "descriptionRow" . }}
`

var baseRepoTemplate = template.Must(
	template.New("repo").
		Funcs(template.FuncMap{
			"join":   strings.Join,
			"wrap":   func(s string) string { return WrapText(s, 80, 2) },
			"cyan":   func(s string) string { return s },
			"blue":   func(s string) string { return s },
			"yellow": func(s string) string { return s },
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

func colour(col, str string) string {
	return col + str + reset
}

func printRepo(w io.Writer, r Repo) error {
	tmpl, err := baseRepoTemplate.Clone()
	if err != nil {
		return err
	}
	if isTTY(w) {
		tmpl = tmpl.Funcs(template.FuncMap{
			"cyan":   func(s string) string { return colour(cyan, s) },
			"blue":   func(s string) string { return colour(blue, s) },
			"yellow": func(s string) string { return colour(yellow, s) },
		})
	}
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
