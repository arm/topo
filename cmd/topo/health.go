package main

import (
	"fmt"
	"os"

	"github.com/arm-debug/topo-cli/internal/health"
	"github.com/arm-debug/topo-cli/internal/output/logger"
	"github.com/arm-debug/topo-cli/internal/output/printable"
	"github.com/arm-debug/topo-cli/internal/output/templates"
	"github.com/spf13/cobra"
)

var healthTarget string

var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "Check the target host environment (container engines, SSH availability)",
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		c, err := GetLogger(cmd)
		c.Log(logger.Entry{
			Level:   logger.Info,
			Message: "woop",
		})
		return fmt.Errorf("noooo")
		sshTarget, err := resolveTarget(healthTarget)
		if err != nil {
			return err
		}
		outputFormat, err := resolveOutput(output)
		if err != nil {
			return err
		}
		report, err := health.Check(sshTarget)
		if err != nil {
			return err
		}
		return printable.Print(templates.PrintableHealthReport(report), os.Stdout, outputFormat)
	},
}

func init() {
	addTargetFlag(healthCmd, &healthTarget)
	rootCmd.AddCommand(healthCmd)
}
