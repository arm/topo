package command

import (
	"fmt"
	"strings"
)

func FormatError(args []string, err error) error {
	cmdStr := strings.Join(args, " ")
	return fmt.Errorf("%s failed: %w", cmdStr, err)
}
