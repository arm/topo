package main

import (
	"os"

	"github.com/arm/topo/internal/output/logger"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
}
