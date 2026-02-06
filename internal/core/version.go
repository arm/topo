package core

import (
	"fmt"
	"strings"
)

// PrintVersion prints version newline-terminated.
func PrintVersion() { fmt.Print(strings.TrimSpace(VersionTxt) + "\n") }
