package main

import (
	"fmt"
	"os"

	"github.com/arm/topo/internal/env"
	"github.com/arm/topo/internal/output/term"
	"github.com/arm/topo/internal/upgrade"
	"github.com/arm/topo/internal/version"
	"github.com/spf13/cobra"
)

const disableSelfUpgradeEnvVar = "TOPO_DISABLE_SELF_UPGRADE"

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade topo to the latest version",
	Long:  "Download and install the latest version of topo from Artifactory, replacing the current binary in place.",
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		outputFormat := resolveOutput(cmd)

		var spinner *term.Spinner
		if outputFormat == term.Plain {
			spinner = term.StartSpinner(os.Stderr, "Upgrading topo...")
		}

		ctx, cancel := contextWithTimeout(cmd)
		defer cancel()

		newVersion, err := upgrade.Upgrade(ctx, spinner)
		if spinner != nil {
			spinner.Stop()
		}
		if err != nil {
			return err
		}

		if newVersion == version.Version {
			fmt.Printf("Already running the latest version of topo (%s)\n", newVersion)
			return nil
		}

		fmt.Printf("Upgraded topo to %s\n", newVersion)
		return nil
	},
}

func init() {
	if isSelfUpgradeDisabled() {
		return
	}
	addTimeoutFlag(upgradeCmd, 0)
	rootCmd.AddCommand(upgradeCmd)
}

func isSelfUpgradeDisabled() bool {
	if env.IsVarTruthy(disableSelfUpgradeEnvVar) {
		return true
	}

	binPath, err := upgrade.CurrentBinaryPath()
	return err == nil && !upgrade.IsBinaryManagedByUs(binPath)
}
