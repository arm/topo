package command

import (
	"fmt"
	"regexp"
)

var binaryRegex = regexp.MustCompile(`^[A-Za-z0-9_+-]+$`)

func ValidateBinaryName(bin string) error {
	if !binaryRegex.MatchString(bin) {
		return fmt.Errorf("%q is not a valid binary name (contains invalid characters)", bin)
	}
	return nil
}
