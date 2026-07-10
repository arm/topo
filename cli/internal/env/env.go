package env

import (
	"os"
	"slices"
	"strings"
)

func IsVarTruthy(name string) bool {
	return slices.Contains(
		[]string{
			"1",
			"true",
			"yes",
			"on",
			"y",
			"enabled",
		},
		strings.ToLower(os.Getenv(name)),
	)
}
