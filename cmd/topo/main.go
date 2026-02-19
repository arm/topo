package main

import (
	"os"

	"github.com/arm/topo/internal/output/console"
	"github.com/arm/topo/internal/output/logger"
	"github.com/arm/topo/internal/output/term"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		outputFormat, outputError := resolveOutput(rootCmd)
		if outputError != nil {
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
