package main

import (
	"os"

	"github.com/arm-debug/topo-cli/internal/health"
	"github.com/arm-debug/topo-cli/internal/output"
	"github.com/spf13/cobra"
)

var (
	healthTarget string
	healthOutput string
)

var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "Check the target host environment (container engines, SSH availability)",
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		sshTarget, err := resolveTarget(healthTarget)
		if err != nil {
			return err
		}
		outputFormat, err := resolveOutput(healthOutput)
		if err != nil {
			return err
		}
		printer := output.NewPrinter(os.Stdout, outputFormat)
		report, err := health.Check(sshTarget)
		if err != nil {
			return err
		}
		return output.PrintHealthReport(printer, report)
	},
}

func init() {
	addTargetFlag(healthCmd, &healthTarget)
	addOutputFlag(healthCmd, &healthOutput)
	rootCmd.AddCommand(healthCmd)
}
