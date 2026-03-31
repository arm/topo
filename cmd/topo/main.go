package main

import (
	"os"

	"github.com/arm/topo/internal/output/logger"
	"github.com/arm/topo/internal/output/term"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		outputFormat, outputError := resolveOutput(rootCmd)
		if outputError != nil {
			outputFormat = term.Plain
		}
		logger.SetOutputFormat(outputFormat)
		logger.Error(err.Error())
		os.Exit(1)
	}
}
