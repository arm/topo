package main

import (
	"os"

	"github.com/arm/topo/internal/ssh"
	"github.com/arm/topo/internal/ssh/sshconfig"
	"github.com/arm/topo/internal/vscode"
	"github.com/spf13/cobra"
)

var getProjectCmd = &cobra.Command{
	Use:    "get-project <compose-filepath>",
	Short:  "Print the project as JSON",
	Hidden: true,
	Args:   cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		composeFilePath := args[0]
		return vscode.PrintProject(os.Stdout, composeFilePath)
	},
}

var setupSSHCommand = &cobra.Command{
	Use:   "setup-ssh",
	Short: "Create a topo-managed SSH config entry for the target",
	Long: `Create a topo-managed SSH config entry for the target in ~/.ssh/topo_config.
	
This will also update the main SSH config (~/.ssh/config) to include the topo-managed configs, if not already included.`,
	Hidden: true,
	Args:   cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true

		targetArg, err := requireTarget(cmd)
		if err != nil {
			return err
		}

		dest := ssh.NewDestination(targetArg)
		targetSlug := dest.Slugify()

		if err != nil {
			return err
		}

		return sshconfig.CreateSSHConfig(targetArg, targetSlug)
	},
}

func init() {
	rootCmd.AddCommand(getProjectCmd)
	addTargetFlag(setupSSHCommand)
	rootCmd.AddCommand(setupSSHCommand)
}
