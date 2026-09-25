package env

import (
	"maps"
	"os"
	"slices"
	"strings"

	"github.com/compose-spec/compose-go/v2/utils"
)

func CurrentValues(paths []string) (map[string]string, error) {
	values, err := ReadFiles(paths)
	if err != nil {
		return nil, err
	}
	maps.Copy(values, utils.GetAsEqualsMap(os.Environ()))
	return values, nil
}

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
