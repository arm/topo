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

const (
	acceptNewHostFlag     = "accept-new-host-keys"
	skipVersionChecksFlag = "skip-version-checks"
)

const skipVersionChecksEnvVar = "TOPO_SKIP_VERSION_CHECKS"

var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "Check the target host environment",
	Long:  "Check the target host environment, including container engines and SSH availability.",
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		outputFormat := resolveOutput(cmd)

		acceptNewHostKeys, err := cmd.Flags().GetBool(acceptNewHostFlag)
		if err != nil {
			panic(fmt.Sprintf("internal error: %s flag not registered: %v", acceptNewHostFlag, err))
		}

		skipVersionCheck := resolveSkipVersionChecks(cmd)

		var spinner *term.Spinner
		if outputFormat == term.Plain {
			spinner = term.StartSpinner(os.Stderr, "Checking health...")
		}

		toPrint := templates.PrintableHealthReport{
			Host: health.CheckHost(health.CheckHostOptions{SkipVersionChecks: skipVersionCheck}),
		}

		if targetArg, ok := lookupTarget(cmd); ok {
			ctx, cancel := contextWithTimeout(cmd)
			defer cancel()
			targetReport, err := health.CheckTarget(ctx, ssh.NewDestination(targetArg), acceptNewHostKeys)
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
	addTimeoutFlag(healthCmd, defaultTimeout)
	healthCmd.Flags().Bool(acceptNewHostFlag, false, "automatically trust and add new SSH host keys for the target")
	healthCmd.Flags().Bool(skipVersionChecksFlag, false, fmt.Sprintf("skip version checks for dependencies (can also be set via %s env var)", skipVersionChecksEnvVar))
	rootCmd.AddCommand(healthCmd)
}

func resolveSkipVersionChecks(cmd *cobra.Command) bool {
	skipVersionCheck, err := cmd.Flags().GetBool(skipVersionChecksFlag)
	if err != nil {
		panic(fmt.Sprintf("internal error: %s flag not registered: %v", skipVersionChecksFlag, err))
	}

	if !skipVersionCheck {
		skipVersionCheck = os.Getenv(skipVersionChecksEnvVar) != ""
	}

	return skipVersionCheck
}
