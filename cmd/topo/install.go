package main

import (
	"os"

	"github.com/arm-debug/topo-cli/internal/install"
	"github.com/arm-debug/topo-cli/internal/output/printable"
	"github.com/arm-debug/topo-cli/internal/output/templates"
	"github.com/arm-debug/topo-cli/internal/ssh"
	"github.com/spf13/cobra"
)

const remoteprocRepoURL = "arm/remoteproc-runtime"

var (
	installRemoteprocTarget string
	installRemoteprocOutput string
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install components to target",
}

var installRemoteprocCmd = &cobra.Command{
	Use:   "remoteproc",
	Short: "Install remoteproc-runtime and shim to a location on the target's PATH",
	Long: `Install remoteproc-runtime and shim to a location on the target's PATH.

Fetches binaries from https://github.com/` + remoteprocRepoURL + `
Set GITHUB_TOKEN to authenticate with the GitHub API and avoid rate limits.

Attempts to replace existing installations if found.
Falls back to ~/bin if no suitable locations are automatically found.
`,
	Args: cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		sshTarget, err := resolveTarget(installRemoteprocTarget)
		if err != nil {
			return err
		}

		outputFormat, err := resolveOutput(installRemoteprocOutput)
		if err != nil {
			return err
		}
		p, err := installRemoteprocRuntime(ssh.Host(sshTarget))
		if err != nil {
			return err
		}
		return printable.Print(p, os.Stdout, outputFormat)
	},
}

func init() {
	rootCmd.AddCommand(installCmd)
	installCmd.AddCommand(installRemoteprocCmd)
	addTargetFlag(installRemoteprocCmd, &installRemoteprocTarget)
	addOutputFlag(installRemoteprocCmd, &installRemoteprocOutput)
}

func installRemoteprocRuntime(targetHost ssh.Host) (printable.Printable, error) {
	results, err := install.InstallBinariesFromGithubRelease(targetHost, remoteprocRepoURL, []string{"remoteproc-runtime", "containerd-shim-remoteproc-v1"})
	if err != nil {
		return nil, err
	}
	return templates.InstallResults(results), nil
}
