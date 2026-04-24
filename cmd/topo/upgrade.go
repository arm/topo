package main

import (
	"fmt"
	"runtime"

	"github.com/arm/topo/internal/output/term"
	"github.com/arm/topo/internal/upgrade"
	"github.com/arm/topo/internal/version"
	"github.com/spf13/cobra"
)

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade topo to the latest version",
	Long:  "Download and install the latest version of topo from Artifactory, replacing the current binary in place.",
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		outputFormat := resolveOutput(cmd)

		ctx, cancel := contextWithTimeout(cmd)
		defer cancel()

		current := version.Version
		var latest string
		err := term.WithSpinner(outputFormat, "Checking for updates...", func() (err error) {
			latest, err = version.FetchLatest(ctx, version.ArtifactoryBaseURL)
			return err
		})
		if err != nil {
			return err
		}

		if current == latest || current == version.Dev {
			fmt.Printf("topo %s is already up to date\n", current)
			return nil
		}

		fmt.Printf("Upgrading topo from %s to %s\n", current, latest)

		binPath, err := upgrade.CurrentBinaryPath()
		if err != nil {
			return fmt.Errorf("failed to determine current binary path: %w", err)
		}

		err = term.WithSpinner(outputFormat, fmt.Sprintf("Downloading topo %s...", latest), func() error {
			downloadURL := upgrade.ArtifactoryDownloadURL(runtime.GOOS, runtime.GOARCH, latest)
			return upgrade.Install(ctx, binPath, downloadURL)
		})
		if err != nil {
			return err
		}

		fmt.Printf("Upgraded topo to %s\n", latest)
		return nil
	},
}

func init() {
	addTimeoutFlag(upgradeCmd, 0)
	rootCmd.AddCommand(upgradeCmd)
}
