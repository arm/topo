package main

import (
	"fmt"
	"os"

	"github.com/arm/topo/internal/setupkeys"
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
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true

		targetArg, err := requireTarget(cmd)
		if err != nil {
			return err
		}

		sshDir, err := ssh.GetConfigDirectory()
		if err != nil {
			return err
		}

		if isLegacyDir, err := ssh.IsLegacyTopoConfigDirectory(sshDir); err != nil {
			return err
		} else if isLegacyDir {
			return fmt.Errorf("legacy topo ssh config entries found; run 'topo migrate-ssh' to migrate to the new single-file format")
		}

		dest := ssh.NewDestination(targetArg)
		user, err := ssh.GetUserFromConfig(dest)
		if err != nil {
			return fmt.Errorf("%w; note: a per user ssh config entry should be created when setting up keys", err)
		}

		dest.User = user
		targetSlug := dest.Slugify()
		if privateKeyPath == "" {
			privateKeyPath, err = setupkeys.GetDefaultPrivateKeyPath(sshDir, targetSlug)
			if err != nil {
				return err
			}
		}

		parsedKeyType, err := setupkeys.ParseKeyType(keyType)
		if err != nil {
			return err
		}

		seq := setupkeys.NewKeySetup(dest, privateKeyPath, parsedKeyType)

		err = seq.Run(os.Stdout)
		if err != nil {
			return err
		}

		modifiers := []ssh.ConfigDirectiveModifier{
			ssh.NewEnsureConfigDirectivePath("IdentityFile", privateKeyPath),
			ssh.NewEnsureConfigDirective("IdentitiesOnly", "yes"),
			ssh.NewEnsureConfigDirective("User", dest.User),
		}

		return ssh.CreateOrModifyConfigFile(sshDir, dest, modifiers)
	},
}

func init() {
	addTargetFlag(setupKeysCmd)
	setupKeysCmd.Flags().StringVar(&privateKeyPath, "key-path", "", "Specify the SSH path where the generated key pair will be stored. Default directory: ~/.ssh. Default public key file name: id_ed25519_topo_<target>.pub)")
	setupKeysCmd.Flags().StringVar(&keyType, "key-type", "ed25519", fmt.Sprintf("Specify the type of SSH key to generate. Supported types: %s, %s. Default: %s", setupkeys.KeyTypeED25519, setupkeys.KeyTypeRSA, setupkeys.KeyTypeED25519))
	rootCmd.AddCommand(setupKeysCmd)
}
