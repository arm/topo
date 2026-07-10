package views

import (
	"strings"
	"text/template"

	"github.com/arm/topo/cli/internal/catalog"
	"github.com/arm/topo/cli/internal/output/term"
)

func getFuncMap(isTTY bool) template.FuncMap {
	f := template.FuncMap{
		"join":              strings.Join,
		"wrap":              func(s string) string { return term.WrapText(s, 80, 2) },
		"cyan":              func(s string) string { return s },
		"blue":              func(s string) string { return s },
		"yellow":            func(s string) string { return s },
		"compatibilityMark": plainCompatibilityMark,
		"cloneCommand":      cloneCommand,
	}

	if isTTY {
		f["cyan"] = func(s string) string { return term.Color(term.Cyan, s) }
		f["blue"] = func(s string) string { return term.Color(term.Blue, s) }
		f["yellow"] = func(s string) string { return term.Color(term.Yellow, s) }
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
