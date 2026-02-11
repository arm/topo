package main

import (
	"os"

	"github.com/arm-debug/topo-cli/internal/output/console"
	"github.com/arm-debug/topo-cli/internal/output/logger"
	"github.com/arm-debug/topo-cli/internal/output/term"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		output, _ := rootCmd.Flags().GetString("output")

		format, err := resolveOutput(output)
		if err != nil {
			format = term.Plain
		}

		c := console.NewLogger(os.Stderr, format)
		c.Log(logger.Entry{
			Level:   logger.Err,
			Message: err.Error(),
		})

		os.Exit(1)
	}
}
