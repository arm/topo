package main

import (
	"fmt"
	"os"

	"github.com/arm/topo/internal/env"
	"github.com/arm/topo/internal/health"
	"github.com/arm/topo/internal/output/term"
	"github.com/arm/topo/internal/output/views"
	"github.com/arm/topo/internal/ssh"
	"github.com/spf13/cobra"
)

const (
	skipVersionChecksFlag = "skip-version-checks"
	verboseFlag           = "verbose"
)

const skipVersionChecksEnvVar = "TOPO_SKIP_VERSION_CHECKS"

var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "Check the target environment",
	Long:  "Check the target environment, including container engines and SSH availability.",
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		outputFormat := resolveOutput(cmd)

		skipVersionCheck := resolveSkipVersionChecks(cmd)
		verbose, err := cmd.Flags().GetBool(verboseFlag)
		if err != nil {
			panic(fmt.Sprintf("internal error: %s flag not registered: %v", verboseFlag, err))
		}
		selectedEngine, err := getSelectedEngine(cmd)
		if err != nil {
			return err
		}

		var spinner *term.Spinner
		if outputFormat == term.Plain {
			spinner = term.StartSpinner(os.Stderr, "Checking health...")
		}

		var target *ssh.Destination
		if targetArg, ok := lookupTarget(cmd); ok {
			destination := ssh.NewDestination(targetArg)
			target = &destination
		}

		ctx, cancel := contextWithTimeout(cmd)
		defer cancel()
		report := health.Check(ctx, health.HealthCheckOptions{
			Engine:                  health.Engine(selectedEngine),
			Target:                  target,
			MissingTargetFixMessage: "provide --target or set TOPO_TARGET to check target health",
			SkipVersionChecks:       skipVersionCheck,
		})

		if spinner != nil {
			spinner.Stop()
		}

		toPrint := views.HealthReportView{
			HealthReport: report,
			Verbose:      verbose,
		}
		return views.Print(toPrint, os.Stdout, outputFormat)
	},
}

func init() {
	addTargetFlag(healthCmd)
	addTimeoutFlag(healthCmd, defaultTimeout)
	if experimentalFeaturesEnabled() {
		addEngineFlag(healthCmd)
	}
	healthCmd.Flags().Bool(skipVersionChecksFlag, false, fmt.Sprintf("skip version checks for dependencies (can also be set via %s env var)", skipVersionChecksEnvVar))
	healthCmd.Flags().BoolP(verboseFlag, "v", false, "show all health checks, including successful checks")
	rootCmd.AddCommand(healthCmd)
}

func resolveSkipVersionChecks(cmd *cobra.Command) bool {
	if env.IsVarTruthy(disableSelfUpgradeEnvVar) {
		return true
	}

	if !cmd.Flags().Changed(skipVersionChecksFlag) {
		return env.IsVarTruthy(skipVersionChecksEnvVar)
	}

	skipVersionChecks, err := cmd.Flags().GetBool(skipVersionChecksFlag)
	if err != nil {
		panic(fmt.Sprintf("internal error: %s flag not registered: %v", skipVersionChecksFlag, err))
	}
	return skipVersionChecks
}
