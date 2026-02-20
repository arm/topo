package main

import (
	"fmt"
	"os"

	"github.com/arm/topo/internal/setup_keys"
	"github.com/spf13/cobra"
)

var setupKeysKeyPath string

var setupKeysCmd = &cobra.Command{
	Use:   "setup-keys",
	Short: "Generate SSH keys for the target and install the public key on the target host",
	Long: `Generate SSH keys for the target and install the public key on the target host.

Use --dry-run to see what commands would be executed without actually running them.`,
	Args: cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true

		dryRun, err := cmd.Flags().GetBool("dry-run")
		if err != nil {
			panic(fmt.Sprintf("internal error: dry-run flag not registered: %v", err))
		}

		resolvedTarget, err := requireTarget(cmd)
		if err != nil {
			return err
		}
		targetFileName := setup_keys.SanitizeTarget(resolvedTarget)
		privKeyPath, err := setup_keys.CreateKeyPair(resolvedTarget, targetFileName, setupKeysKeyPath, os.Stdout, dryRun)
		if err != nil {
			return err
		}

		return setup_keys.TransferPubKey(resolvedTarget, privKeyPath, os.Stdout, dryRun)
	},
}

func init() {
	addTargetFlag(setupKeysCmd)
	addDryRunFlag(setupKeysCmd)
	setupKeysCmd.Flags().StringVar(&setupKeysKeyPath, "key-path", "", "Specify the SSH path where the generated key pair will be stored. Default directory: ~/.ssh. Default public key file name: id_ed25519_topo_<target>.pub)")
	rootCmd.AddCommand(setupKeysCmd)
}
