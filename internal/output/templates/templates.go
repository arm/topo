package templates

import (
	"strings"
	"text/template"

	"github.com/arm/topo/internal/catalog"
	"github.com/arm/topo/internal/output/term"
)

func getFuncMap(isTTY bool) template.FuncMap {
	f := template.FuncMap{
		"join":              strings.Join,
		"wrap":              func(s string) string { return term.WrapText(s, 80, 2) },
		"cyan":              func(s string) string { return s },
		"blue":              func(s string) string { return s },
		"yellow":            func(s string) string { return s },
		"compatibilityMark": plainCompatibilityMark,
	}

	if isTTY {
		f["cyan"] = func(s string) string { return term.Color(term.Cyan, s) }
		f["blue"] = func(s string) string { return term.Color(term.Blue, s) }
		f["yellow"] = func(s string) string { return term.Color(term.Yellow, s) }
	}

	return f
}

func plainCompatibilityMark(c *catalog.Compatibility) string {
	if c == nil {
		return ""
	}
	if c.Supported {
		return "✅"
	}
	return "❌"
}
