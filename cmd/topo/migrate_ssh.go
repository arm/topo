package main

import (
	"github.com/arm/topo/internal/ssh"
	"github.com/spf13/cobra"
)

var migrateSSHCmd = &cobra.Command{
	Use:    "migrate-ssh",
	Short:  "Migrate from the legacy multi-file SSH config to the new single-file format",
	Hidden: true,
	Args:   cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		return ssh.MigrateLegacyTopoConfig()
	},
}

func init() {
	rootCmd.AddCommand(migrateSSHCmd)
}
