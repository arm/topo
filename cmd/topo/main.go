package main

import (
	"fmt"
	"os"

	"github.com/arm-debug/topo-cli/internal/run"
)

func main() {
	if err := run.Execute(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
