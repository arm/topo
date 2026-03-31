package main

import (
	"os"

	"github.com/arm/topo/internal/output/logger"
)

func main() {
	outputFormat := resolveOutput(rootCmd)
	logger.SetOutputFormat(outputFormat)

	if err := rootCmd.Execute(); err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
}
