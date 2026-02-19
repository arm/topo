package main

import (
	"fmt"
	"os"

	"github.com/arm/topo/internal/health"
	"github.com/arm/topo/internal/output/printable"
	"github.com/arm/topo/internal/output/templates"
	"github.com/spf13/cobra"
)

const acceptNewHostFlag = "accept-new-host-keys"

var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "Check the target host environment (container engines, SSH availability)",
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		sshTarget, err := requireTarget(cmd)
		if err != nil {
			return err
		}
		outputFormat, err := resolveOutput(cmd)
		if err != nil {
			return err
		}

		acceptNewHostKeys, err := cmd.Flags().GetBool(acceptNewHostFlag)
		if err != nil {
			panic(fmt.Sprintf("internal error: %s flag not registered: %v", acceptNewHostFlag, err))
		}

		report, err := health.Check(sshTarget, acceptNewHostKeys)
		if err != nil {
			return err
		}
		return printable.Print(templates.PrintableHealthReport(report), os.Stdout, outputFormat)
	},
}

func init() {
	addTargetFlag(healthCmd)
	healthCmd.Flags().Bool(acceptNewHostFlag, false, "Automatically trust and add new SSH host keys for the target")
	rootCmd.AddCommand(healthCmd)
}
