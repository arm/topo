package views

import (
	"strings"
	"text/template"

	"github.com/arm/topo/internal/catalog"
	"github.com/arm/topo/internal/output/term"
)

func getFuncMap(palette term.Palette) template.FuncMap {
	f := template.FuncMap{
		"join":              strings.Join,
		"wrap":              func(s string) string { return term.WrapText(s, 80, 2) },
		"cyan":              func(s string) string { return palette.Color(term.Cyan, s) },
		"blue":              func(s string) string { return palette.Color(term.Blue, s) },
		"yellow":            func(s string) string { return palette.Color(term.Yellow, s) },
		"compatibilityMark": plainCompatibilityMark,
		"cloneCommand":      cloneCommand,
	}

	return f
}

func plainCompatibilityMark(c catalog.CompatibilityStatus) string {
	if c == catalog.CompatibilitySupported {
		return "✅"
	}
	if c == catalog.CompatibilityUnsupported {
		return "❌"
	}
	return ""
}

func cloneCommand(project catalog.ProjectWithCompatibility) string {
	source := project.URL
	if project.Ref != "" {
		source += "#" + project.Ref
	}
	return "topo clone " + source
}
