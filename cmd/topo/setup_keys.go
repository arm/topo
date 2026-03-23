package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/arm/topo/internal/setupkeys"
	"github.com/arm/topo/internal/setupkeys/sshconfig"
	"github.com/arm/topo/internal/ssh"
	"github.com/spf13/cobra"
)

var (
	privateKeyPath string
	keyType        string
)

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

		targetArg, err := requireTarget(cmd)
		if err != nil {
			return err
		}

		targetSlug := ssh.NewConfig(targetArg).Slugify()
		if privateKeyPath == "" {
			privateKeyPath, err = setupkeys.GetDefaultPrivateKeyPath(targetSlug)
			if err != nil {
				return err
			}
		}

		parsedKeyType, err := setupkeys.ParseKeyType(keyType)
		if err != nil {
			return err
		}

		cfg := ssh.NewConfig(targetArg)
		seq, err := setupkeys.NewKeySetup(cfg.Destination, privateKeyPath, parsedKeyType)
		if err != nil {
			return err
		}

		if dryRun {
			err = seq.DryRun(os.Stdout)
		} else {
			err = seq.Run(os.Stdout)
		}

		if err != nil {
			return err
		}

		directives := []sshconfig.SSHConfigDirective{
			{Key: "IdentityFile", Value: filepath.ToSlash(privateKeyPath)},
			{Key: "IdentitiesOnly", Value: "yes"},
		}

		return sshconfig.ModifySSHConfig(targetArg, targetSlug, dryRun, os.Stdout, directives)
	},
}

func init() {
	addTargetFlag(setupKeysCmd)
	addDryRunFlag(setupKeysCmd)
	setupKeysCmd.Flags().StringVar(&privateKeyPath, "key-path", "", "Specify the SSH path where the generated key pair will be stored. Default directory: ~/.ssh. Default public key file name: id_ed25519_topo_<target>.pub)")
	setupKeysCmd.Flags().StringVar(&keyType, "key-type", "ed25519", fmt.Sprintf("Specify the type of SSH key to generate. Supported types: %s, %s. Default: %s", setupkeys.KeyTypeED25519, setupkeys.KeyTypeRSA, setupkeys.KeyTypeED25519))
	rootCmd.AddCommand(setupKeysCmd)
}
