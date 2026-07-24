package main

import (
	"context"
	"os"

	"github.com/arm/topo/internal/install"
	"github.com/arm/topo/internal/output/views"
	"github.com/arm/topo/internal/runner"
	"github.com/arm/topo/internal/ssh"
	"github.com/spf13/cobra"
)

const (
	remoteprocRuntimeURL         = "https://artifacts.tools.arm.com/remoteproc-runtime/"
	remoteprocRuntimeArchiveName = "remoteproc-runtime_%s_linux_arm64.tar.gz"
)

var installRemoteprocRuntimeCmd = &cobra.Command{
	Use:   "remoteproc-runtime",
	Short: "Install remoteproc-runtime and shim to a location on the target's PATH",
	Long: `Install remoteproc-runtime and shim to a location on the target's PATH.

Attempts to replace existing installations if found.
Falls back to ~/bin if no suitable locations are automatically found.`,
	Args: cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		targetArg, err := requireTarget(cmd)
		if err != nil {
			return err
		}

		outputFormat := resolveOutput(cmd)
		p, err := installRemoteprocRuntime(cmd.Context(), ssh.NewDestination(targetArg))
		if err != nil {
			return err
		}
		return views.Print(p, os.Stdout, outputFormat)
	},
}

func installRemoteprocRuntime(ctx context.Context, targetDest ssh.Destination) (views.View, error) {
	r := runner.For(targetDest)
	results, err := install.InstallBinariesFromArtifactory(ctx, r, remoteprocRuntimeURL, remoteprocRuntimeArchiveName, []string{"remoteproc-runtime", "containerd-shim-remoteproc-v1"})
	if err != nil {
		return nil, err
	}
	return views.InstallResults(results), nil
}

func init() {
	installCmd.AddCommand(installRemoteprocRuntimeCmd)
	addTargetFlag(installRemoteprocRuntimeCmd)
}
