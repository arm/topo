package main

import (
	"fmt"
	"os"

	"github.com/arm/topo/internal/health"
	"github.com/arm/topo/internal/output/printable"
	"github.com/arm/topo/internal/output/templates"
	"github.com/arm/topo/internal/output/term"
	"github.com/arm/topo/internal/ssh"
	"github.com/spf13/cobra"
)

const acceptNewHostFlag = "accept-new-host-keys"

var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "Check the target host environment (container engines, SSH availability)",
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		outputFormat, err := resolveOutput(cmd)
		if err != nil {
			return err
		}

		acceptNewHostKeys, err := cmd.Flags().GetBool(acceptNewHostFlag)
		if err != nil {
			panic(fmt.Sprintf("internal error: %s flag not registered: %v", acceptNewHostFlag, err))
		}

		var spinner *term.Spinner
		if outputFormat == term.Plain {
			spinner = term.StartSpinner(os.Stderr, "Checking health...")
		}

		toPrint := templates.PrintableHealthReport{
			Host: health.CheckHost(),
		}

		if targetArg, ok := lookupTarget(cmd); ok {
			targetReport, err := health.CheckTarget(ssh.NewDestination(targetArg), acceptNewHostKeys, sshConnectTimeout)
			if err != nil {
				if spinner != nil {
					spinner.Stop()
				}
				return err
			}
			toPrint.Target = &targetReport
		} else {
			toPrint.TargetHint = "provide --target or set TOPO_TARGET to check target health"
		}

		if spinner != nil {
			spinner.Stop()
		}

		return printable.Print(toPrint, os.Stdout, outputFormat)
	},
}

func init() {
	addTargetFlag(healthCmd)
	healthCmd.Flags().Bool(acceptNewHostFlag, false, "Automatically trust and add new SSH host keys for the target")
	rootCmd.AddCommand(healthCmd)
}
