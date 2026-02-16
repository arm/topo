package main

import (
	"os"

	"github.com/arm-debug/topo-cli/internal/output/console"
	"github.com/arm-debug/topo-cli/internal/output/logger"
	"github.com/arm-debug/topo-cli/internal/output/term"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		outputFormat, outputFormatError := resolveOutput(rootCmd)
		if outputFormatError != nil {
			outputFormat = term.Plain
		}
		c := console.NewLogger(os.Stderr, outputFormat)
		c.Log(logger.Entry{
			Level:   logger.Err,
			Message: err.Error(),
		})

		os.Exit(1)
	}
}
