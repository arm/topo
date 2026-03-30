package main

import (
	"os"

	"github.com/arm/topo/internal/install"
	"github.com/arm/topo/internal/output/printable"
	"github.com/arm/topo/internal/output/templates"
	"github.com/arm/topo/internal/ssh"
	"github.com/arm/topo/internal/target"
	"github.com/spf13/cobra"
)

const remoteprocRuntimeRepoURL = "arm/remoteproc-runtime"

var installRemoteprocRuntimeCmd = &cobra.Command{
	Use:   "remoteproc-runtime",
	Short: "Install remoteproc-runtime and shim to a location on the target's PATH",
	Long: `Install remoteproc-runtime and shim to a location on the target's PATH.

Fetches binaries from https://github.com/` + remoteprocRuntimeRepoURL + `
Set GITHUB_TOKEN to authenticate with the GitHub API and avoid rate limits.

Attempts to replace existing installations if found.
Falls back to ~/bin if no suitable locations are automatically found.
`,
	Args: cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		targetArg, err := requireTarget(cmd)
		if err != nil {
			return err
		}

		outputFormat, err := resolveOutput(cmd)
		if err != nil {
			return err
		}
		p, err := installRemoteprocRuntime(ssh.NewDestination(targetArg))
		if err != nil {
			return err
		}
		return printable.Print(p, os.Stdout, outputFormat)
	},
}

func installRemoteprocRuntime(targetDest ssh.Destination) (printable.Printable, error) {
	conn := target.NewConnection(targetDest, target.ConnectionOptions{})
	results, err := install.InstallBinariesFromGithubRelease(&conn, remoteprocRuntimeRepoURL, []string{"remoteproc-runtime", "containerd-shim-remoteproc-v1"})
	if err != nil {
		return nil, err
	}
	return templates.InstallResults(results), nil
}

func init() {
	installCmd.AddCommand(installRemoteprocRuntimeCmd)
	addTargetFlag(installRemoteprocRuntimeCmd)
}
