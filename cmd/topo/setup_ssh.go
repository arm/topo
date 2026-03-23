package main

import (
	"os"

	"github.com/arm/topo/internal/ssh"
	"github.com/arm/topo/internal/ssh/sshconfig"
	"github.com/spf13/cobra"
)

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

		targetSlug := ssh.NewConfig(targetArg).Slugify()

		if err != nil {
			return err
		}

		return sshconfig.ModifySSHConfig(targetArg, targetSlug, false, os.Stdout, nil)
	},
}

func init() {
	addTargetFlag(setupSSHCommand)
	rootCmd.AddCommand(setupSSHCommand)
}
